// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 mp0rta and mqvpn contributors

package com.mqvpn.sdk.core.internal

import android.net.Network
import android.system.OsConstants
import android.util.Log
import com.mqvpn.sdk.core.MqvpnTunnel
import com.mqvpn.sdk.native_.NativeBridge
import com.mqvpn.sdk.network.NetworkEvent
import com.mqvpn.sdk.network.NetworkMonitor
import com.mqvpn.sdk.network.NetworkPath
import com.mqvpn.sdk.network.PathBinder
import com.mqvpn.sdk.runtime.ExecutorStoppedException
import com.mqvpn.sdk.runtime.MqvpnExecutor
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext

/**
 * Bridges [NetworkEvent]s from NetworkMonitor to libmqvpn path management.
 *
 * Kotlin owns the sockets (creates, pins, closes); the native reactor owns
 * their I/O (bind ctx, poll, drain on the engine thread). Mobile lifecycle:
 * remove + add, never drop/reactivate.
 *
 * Execution context:
 * - handleEvent() runs on IO dispatcher (caller provides)
 * - Socket creation (blocking I/O) happens on calling thread
 * - tunnel.addPath/connect/removePath/pathReleased are serialized via executor.call
 * - handleBadFd() is called by the executor ON the engine thread
 */
internal class PathManager(
    private val executor: MqvpnExecutor,
    private val tunnel: MqvpnTunnel,
    private val networkMonitor: NetworkMonitor,
    private val protector: (Int) -> Boolean,
    private val serverHost: String,
    private val serverPort: Int,
    /** Ledger corruption (REACTOR_POISONED): the session must end. */
    private val onFatal: (String) -> Unit,
    private val bindUdp: (Network, String, Int, (Int) -> Boolean) -> Int =
        { network, host, port, prot ->
            PathBinder.bindAndDetachUdp(network, host, port, prot)
        },
    private val closeFd: (Int) -> Unit = ::closeFdSafe,
    /**
     * tailbond: whether an available network should carry a path right now
     * (per-path enable, priority/standby, data cap). A network the gate
     * refuses stays known to the monitor and is attached by a later
     * [reconcile] once the gate lets it through; an attached one the gate
     * now refuses is detached the same way.
     */
    private val wanted: (NetworkPath) -> Boolean = { true },
) {
    // tailbond: events and reconciles both attach/detach; one at a time.
    private val mutex = Mutex()
    private var connected = false
    private val pathHandles = mutableMapOf<Network, Long>()  // network → pathHandle
    private val pathFds = mutableMapOf<Long, Int>()           // pathHandle → fd

    /**
     * Handle a network event. Must be called from IO dispatcher.
     *
     * Available: [IO] bindSocket → [engine] addPath (+ connect once)
     * Lost: [engine] removePath → close(fd) → pathReleased
     */
    suspend fun handleEvent(event: NetworkEvent) = mutex.withLock {
        when (event) {
            is NetworkEvent.Available ->
                if (wanted(event.path)) attach(event.path)
                else Log.i(TAG, "Path held back by policy: ${event.path.name}")
            is NetworkEvent.Lost -> detach(event.path)
        }
    }

    /**
     * tailbond: bring the attached set in line with [wanted]: attach every
     * available network it accepts, detach every attached one it refuses.
     * Called from IO when settings or path health change.
     */
    suspend fun reconcile() = mutex.withLock {
        val available = networkMonitor.activeNetworks.values.toList()
        // pathHandles is engine-thread state
        val attached = handOver { pathHandles.keys.toSet() } ?: return@withLock
        for (path in available) {
            val want = wanted(path)
            if (want && path.network !in attached) {
                Log.i(TAG, "Policy attaches ${path.name}")
                attach(path)
            } else if (!want && path.network in attached) {
                Log.i(TAG, "Policy detaches ${path.name}")
                detach(path)
            }
        }
    }

    private suspend fun attach(path: NetworkPath) {
        val network = path.network
        val name = path.name

        // Step 1: Create socket (blocking I/O, runs on IO thread)
        val fd = bindUdp(network, serverHost, serverPort, protector)
        if (fd < 0) {
            Log.e(TAG, "Failed to bind socket for $name, will retry on next event")
            networkMonitor.removeNetwork(network)
            return
        }

        // Step 2: Add path + connect on the engine thread. The fd is ours
        // until the block registers it (null = the hand-over was refused).
        val handle = handOver {
            // onLost can fire on a binder thread while bind was running on IO.
            // If the network is no longer in NetworkMonitor's active set, abort
            // — adding a path bound to an already-dead Network would leak a slot
            // (Lost wouldn't fire again for this Network).
            if (!networkMonitor.activeNetworks.containsKey(network)) {
                Log.i(TAG, "Network $name lost during bind, discarding fd")
                return@handOver ABORT_LOST_DURING_BIND
            }
            val h = tunnel.addPath(fd, name)
            if (h < 0) {
                Log.e(TAG, "addPath failed for $name: $h")
                return@handOver h
            }
            pathHandles[network] = h
            pathFds[h] = fd

            if (!connected) {
                tunnel.setServerAddr(serverHost, serverPort)
                tunnel.connect()
                connected = true
            }
            h
        }

        if (handle == null || handle < 0) {
            if (handle == null) Log.w(TAG, "executor stopped during the bind of $name; closing fd=$fd")
            closeFd(fd) // never registered: still ours
            return
        }
        Log.i(TAG, "Path added: $name (handle=$handle, fd=$fd)")
    }

    private suspend fun detach(path: NetworkPath) {
        val network = path.network
        val name = path.name

        // One engine-thread block: remove (library abandons the path, the
        // reactor stops polling) → close (ours) → released (the library
        // finalises the transport). Same thread, same block: no fd-reuse race.
        // A refused hand-over means the session's cleanup closes every fd.
        val handle = handOver {
            val h = pathHandles.remove(network) ?: return@handOver -1L
            val fd = pathFds.remove(h)
            val rc = tunnel.removePath(h)
            if (rc != 0) Log.w(TAG, "removePath for $name returned $rc")
            if (fd != null) closeFd(fd)
            checkReleased(tunnel.pathReleased(h), name)
            h
        } ?: run {
            Log.w(TAG, "executor stopped; the cleanup closes $name's fd")
            return
        }

        if (handle < 0) return
        Log.i(TAG, "Path removed: $name (handle=$handle)")
    }

    /**
     * Engine-thread hand-over of fd ownership. Runs [block] to completion or
     * is refused up front — never cancelled mid-flight: service shutdown
     * cancels the IO scope while binds are in flight, and a skipped block
     * would leave an fd nobody closes (or, after a registration, one we must
     * not close). Null = the executor stopped before the block ran.
     */
    private suspend fun <T> handOver(block: () -> T): T? =
        try {
            withContext(NonCancellable) { executor.call(block) }
        } catch (e: ExecutorStoppedException) {
            null // only the refusal; an exception from inside the block propagates
        }

    /**
     * The reactor found the path's fd closed behind us (POLLNVAL). Engine
     * thread (called by the executor's onBadFd). The number may already
     * belong to another socket, so it is dropped from the ledger WITHOUT a
     * close. No Lost comes while the Network stays up, so it also leaves the
     * monitor's set, as on a bind failure: the next capability update
     * re-adds the path with a fresh fd.
     */
    fun handleBadFd(handle: Long) {
        val network = pathHandles.entries.firstOrNull { it.value == handle }?.key
        if (network != null) pathHandles.remove(network)
        val fd = pathFds.remove(handle)
        Log.e(TAG, "Path fd closed behind the platform (handle=$handle, fd=$fd); removing without close")
        val rc = tunnel.removePath(handle)
        if (rc != 0) Log.w(TAG, "removePath for handle=$handle returned $rc")
        checkReleased(tunnel.pathReleased(handle), "handle=$handle")
        if (network != null) networkMonitor.removeNetwork(network)
    }

    internal fun checkReleased(rc: Int, what: String) {
        when (rc) {
            0 -> {}
            NativeBridge.REACTOR_POISONED ->
                onFatal("path release for $what: ledger corruption (poisoned)")
            else -> Log.w(TAG, "pathReleased for $what returned $rc; transport stays library-owned")
        }
    }

    /** Close all remaining fds. Called from the engine thread during cleanup, AFTER tunnel.destroy(). */
    fun closeAllFds() {
        for ((_, fd) in pathFds) {
            closeFd(fd)
        }
        pathFds.clear()
        pathHandles.clear()
        connected = false
    }

    companion object {
        private const val TAG = "PathManager"

        // An untagged close(2) through the JNI; NativeBridge.closeFd says why.
        private fun closeFdSafe(fd: Int) {
            val err = NativeBridge.closeFd(fd)
            if (err != 0) Log.w(TAG, "close fd=$fd failed: ${OsConstants.errnoName(err) ?: err}")
        }

        /**
         * Sentinel returned from the executor block in [attach] when
         * the network was lost during the IO bind step. Must not collide with
         * any negative error code returned by [MqvpnTunnel.addPath] (xquic /
         * JNI errors are small negatives in the 0..-32 range).
         */
        private const val ABORT_LOST_DURING_BIND = -1000L
    }
}
