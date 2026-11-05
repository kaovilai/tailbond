package dev.tailbond.app.ble

import android.Manifest
import android.annotation.SuppressLint
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothGatt
import android.bluetooth.BluetoothGattCharacteristic
import android.bluetooth.BluetoothGattDescriptor
import android.bluetooth.BluetoothGattServer
import android.bluetooth.BluetoothGattServerCallback
import android.bluetooth.BluetoothGattService
import android.bluetooth.BluetoothManager
import android.bluetooth.BluetoothProfile
import android.bluetooth.BluetoothStatusCodes
import android.bluetooth.le.AdvertiseCallback
import android.bluetooth.le.AdvertiseData
import android.bluetooth.le.AdvertiseSettings
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.os.ParcelUuid
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import androidx.core.content.getSystemService
import dev.tailbond.app.R
import dev.tailbond.app.data.Repository
import dev.tailbond.app.share.HubService
import dev.tailbond.app.share.HubState
import dev.tailbond.app.share.ShareProtocol
import dev.tailbond.app.ui.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import java.io.ByteArrayOutputStream
import java.security.SecureRandom
import java.util.ArrayDeque
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors

/** "One click from your Mac", as the Share screen shows it. */
enum class MacLinkStatus { OFF, WAITING, CONNECTED, NO_BLUETOOTH, NO_PERMISSION, ERROR }

data class MacLinkLive(
    val status: MacLinkStatus = MacLinkStatus.OFF,
    /** Logged-in Macs ("tiger-mbp"). */
    val macs: List<String> = emptyList(),
    val shizuku: ShizukuStatus = ShizukuStatus.NOT_RUNNING,
    val error: String = "",
)

object MacLinkState {
    private val _live = MutableStateFlow(MacLinkLive())
    val live: StateFlow<MacLinkLive> = _live.asStateFlow()
    fun set(f: (MacLinkLive) -> MacLinkLive) {
        while (true) {
            val cur = _live.value
            if (_live.compareAndSet(cur, f(cur))) return
        }
    }
}

/**
 * "One click from your Mac" (docs/phone-hub.md): a Bluetooth LE GATT
 * server a Mac paired with the same home server connects to, to start this
 * phone's hotspot (Shizuku) and "Share bonded connection". The protocol is
 * [BleProtocol]; the logic, testable without Android, is [BleServer] and
 * [BleSession]; this service is the Android glue: the GATT server,
 * advertising, notifications one at a time, a foreground notification.
 */
@SuppressLint("MissingPermission") // checked in hasPermissions() before anything Bluetooth
class BleControlService : Service() {
    private lateinit var repo: Repository
    private val io: ExecutorService = Executors.newSingleThreadExecutor { Thread(it, "ble-control") }
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private var gatt: BluetoothGattServer? = null
    private var hotspot: ShizukuHotspot? = null
    private lateinit var server: BleServer
    private val devices = HashMap<String, BluetoothDevice>()
    private val prepared = HashMap<String, ByteArrayOutputStream>()
    private val queue = ArrayDeque<Pair<String, ByteArray>>()
    private var sending = false
    private val random = SecureRandom()

    override fun onBind(intent: Intent?): IBinder? = null

    /** Work on [io]; an exception is logged, never kills the app (and its VPN). */
    private fun onIo(block: () -> Unit) = io.execute {
        try {
            block()
        } catch (e: Exception) {
            Log.w(TAG, "ble: ${e.javaClass.simpleName}: ${e.message}")
        }
    }

    /** Bluetooth turned on later: open the GATT server then; off: close it. */
    private var btWatching = false
    private val btReceiver = object : BroadcastReceiver() {
        override fun onReceive(c: Context, i: Intent) {
            when (i.getIntExtra(BluetoothAdapter.EXTRA_STATE, -1)) {
                BluetoothAdapter.STATE_ON -> if (gatt == null) open()
                BluetoothAdapter.STATE_TURNING_OFF, BluetoothAdapter.STATE_OFF -> closeGatt()
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        repo = Repository.get(this)
        getSystemService<NotificationManager>()?.createNotificationChannel(
            NotificationChannel(CHANNEL, getString(R.string.mac_link_channel), NotificationManager.IMPORTANCE_LOW),
        )
        val hs = ShizukuHotspot(this).also { hotspot = it }
        val control = AppPhoneControl(this, hs, object : HubControl {
            override fun start() {
                HubService.start(this@BleControlService)
            }
            override fun stop() = HubService.stop(this@BleControlService)
            override fun live() = HubState.live.value
        })
        server = BleServer(object : GattPeer {
            override fun send(device: String, piece: ByteArray) = enqueue(device, piece)
            override fun disconnect(device: String) {
                devices[device]?.let { d -> gatt?.cancelConnection(d) }
            }
        }) { BleSession(::identity, control, { n -> ByteArray(n).also(random::nextBytes) }, { Log.i(TAG, it) }) }
    }

    /** This phone to a Mac: its hub's id and label, its share key id and the login check. */
    private fun identity(): Identity? {
        val server = repo.servers.value.firstOrNull() ?: return null
        val key = repo.shareKey(server.payload)?.toByteArray(Charsets.UTF_8) ?: return null
        return Identity(
            id = HubService.hubId(repo), label = HubService.label(repo), keyId = ShareProtocol.keyId(key),
            verify = BleProtocol.verifier(key, allowed = { hub -> repo.settings.value.shareAllowedHubs.let { it.isEmpty() || hub in it } }),
        )
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stop()
            return START_NOT_STICKY
        }
        if (!hasPermissions(this)) {
            MacLinkState.set { MacLinkLive(status = MacLinkStatus.NO_PERMISSION) }
            stopSelf()
            return START_NOT_STICKY
        }
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID, notification(getString(R.string.mac_link_waiting)),
            if (Build.VERSION.SDK_INT >= 29) ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE else 0,
        )
        if (!btWatching) {
            btWatching = true
            ContextCompat.registerReceiver(this, btReceiver, IntentFilter(BluetoothAdapter.ACTION_STATE_CHANGED), ContextCompat.RECEIVER_NOT_EXPORTED)
        }
        if (gatt == null) open()
        if (!pushing) {
            pushing = true
            // the hub's state, pushed to logged-in Macs
            scope.launch {
                HubState.live.map { it.status }.distinctUntilChanged().collect { st ->
                    onIo { server.push(mapOf("hub" to JsonPrimitive(st.name.lowercase()))) }
                }
            }
        }
        return START_STICKY
    }

    private var pushing = false

    private fun open() {
        val bm = getSystemService<BluetoothManager>()
        val adapter = bm?.adapter
        if (adapter == null || !adapter.isEnabled) {
            MacLinkState.set { it.copy(status = MacLinkStatus.NO_BLUETOOTH) }
            return
        }
        val service = BluetoothGattService(BleProtocol.SERVICE, BluetoothGattService.SERVICE_TYPE_PRIMARY).apply {
            addCharacteristic(BluetoothGattCharacteristic(
                BleProtocol.RX,
                BluetoothGattCharacteristic.PROPERTY_WRITE or BluetoothGattCharacteristic.PROPERTY_WRITE_NO_RESPONSE,
                BluetoothGattCharacteristic.PERMISSION_WRITE,
            ))
            addCharacteristic(BluetoothGattCharacteristic(
                BleProtocol.TX, BluetoothGattCharacteristic.PROPERTY_NOTIFY, BluetoothGattCharacteristic.PERMISSION_READ,
            ).apply {
                addDescriptor(BluetoothGattDescriptor(BleProtocol.CCCD, BluetoothGattDescriptor.PERMISSION_READ or BluetoothGattDescriptor.PERMISSION_WRITE))
            })
        }
        gatt = bm.openGattServer(this, callback)?.apply { addService(service) }
        if (gatt == null) {
            MacLinkState.set { it.copy(status = MacLinkStatus.ERROR, error = "Bluetooth refused the GATT server") }
            return
        }
        advertise()
        MacLinkState.set { it.copy(status = MacLinkStatus.WAITING, error = "", shizuku = hotspot?.shizuku() ?: ShizukuStatus.NOT_RUNNING) }
    }

    private val advertiseCallback = object : AdvertiseCallback() {
        override fun onStartFailure(errorCode: Int) {
            Log.w(TAG, "ble: advertising failed ($errorCode)")
            MacLinkState.set { it.copy(status = MacLinkStatus.ERROR, error = "Bluetooth advertising failed ($errorCode)") }
        }
    }

    /** Connectable advertising stops on a connection on some phones: started again on every change. */
    private fun advertise() {
        val adv = getSystemService<BluetoothManager>()?.adapter?.bluetoothLeAdvertiser ?: return
        runCatching { adv.stopAdvertising(advertiseCallback) }
        adv.startAdvertising(
            AdvertiseSettings.Builder()
                .setAdvertiseMode(AdvertiseSettings.ADVERTISE_MODE_BALANCED)
                .setTxPowerLevel(AdvertiseSettings.ADVERTISE_TX_POWER_MEDIUM)
                .setConnectable(true)
                .setTimeout(0)
                .build(),
            AdvertiseData.Builder().addServiceUuid(ParcelUuid(BleProtocol.SERVICE)).setIncludeDeviceName(false).build(),
            advertiseCallback,
        )
    }

    /** Notifications go out one at a time (onNotificationSent). On [io]. */
    private fun enqueue(device: String, piece: ByteArray) {
        queue.add(device to piece)
        if (!sending) sendNext()
    }

    private fun sendNext() {
        val g = gatt ?: return queue.clear()
        val tx = g.getService(BleProtocol.SERVICE)?.getCharacteristic(BleProtocol.TX) ?: return
        while (queue.isNotEmpty()) {
            val (addr, piece) = queue.poll()!!
            val d = devices[addr] ?: continue
            sending = true
            val ok = if (Build.VERSION.SDK_INT >= 33) {
                g.notifyCharacteristicChanged(d, tx, false, piece) == BluetoothStatusCodes.SUCCESS
            } else {
                @Suppress("DEPRECATION")
                tx.value = piece
                @Suppress("DEPRECATION")
                g.notifyCharacteristicChanged(d, tx, false)
            }
            if (ok) return
            sending = false
        }
        sending = false
    }

    private fun refresh() {
        val macs = server.macs
        MacLinkState.set {
            it.copy(
                status = if (server.connected > 0) MacLinkStatus.CONNECTED else MacLinkStatus.WAITING,
                macs = macs, shizuku = hotspot?.shizuku() ?: ShizukuStatus.NOT_RUNNING,
            )
        }
        val text = if (macs.isEmpty()) getString(R.string.mac_link_waiting) else getString(R.string.mac_link_connected, macs.joinToString(", "))
        getSystemService<NotificationManager>()?.notify(NOTIFICATION_ID, notification(text))
    }

    private val callback = object : BluetoothGattServerCallback() {
        override fun onConnectionStateChange(device: BluetoothDevice, status: Int, newState: Int) {
            val addr = device.address
            onIo {
                if (newState == BluetoothProfile.STATE_CONNECTED) {
                    devices[addr] = device
                    server.connected(addr)
                } else {
                    devices.remove(addr)
                    prepared.remove(addr)
                    server.disconnected(addr)
                }
                refresh()
            }
            scope.launch { runCatching { advertise() } }
        }

        override fun onMtuChanged(device: BluetoothDevice, mtu: Int) {
            onIo { server.mtu(device.address, mtu) }
        }

        override fun onNotificationSent(device: BluetoothDevice, status: Int) {
            onIo {
                sending = false
                sendNext()
            }
        }

        override fun onCharacteristicWriteRequest(
            device: BluetoothDevice, requestId: Int, ch: BluetoothGattCharacteristic,
            preparedWrite: Boolean, responseNeeded: Boolean, offset: Int, value: ByteArray,
        ) {
            val ok = ch.uuid == BleProtocol.RX
            if (responseNeeded) {
                gatt?.sendResponse(device, requestId, if (ok) BluetoothGatt.GATT_SUCCESS else BluetoothGatt.GATT_WRITE_NOT_PERMITTED, offset, if (preparedWrite) value else null)
            }
            if (!ok) return
            val addr = device.address
            onIo {
                if (preparedWrite) {
                    // a long write: at most one framed message, pieces in order
                    val buf = prepared.getOrPut(addr) { ByteArrayOutputStream() }
                    if (offset != buf.size() || buf.size() + value.size > BleProtocol.MAX_MESSAGE + 2) {
                        prepared.remove(addr)
                        server.disconnected(addr)
                        gatt?.cancelConnection(device)
                    } else {
                        buf.write(value)
                    }
                } else {
                    server.write(addr, value)
                    refresh()
                }
            }
        }

        override fun onExecuteWrite(device: BluetoothDevice, requestId: Int, execute: Boolean) {
            gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, 0, null)
            val addr = device.address
            onIo {
                val buf = prepared.remove(addr)
                if (execute && buf != null) {
                    server.write(addr, buf.toByteArray())
                    refresh()
                }
            }
        }

        override fun onDescriptorWriteRequest(
            device: BluetoothDevice, requestId: Int, d: BluetoothGattDescriptor,
            preparedWrite: Boolean, responseNeeded: Boolean, offset: Int, value: ByteArray,
        ) {
            if (responseNeeded) gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, null)
        }
    }

    /** The GATT server and advertising off (Bluetooth going off, or stop). */
    private fun closeGatt() {
        runCatching { getSystemService<BluetoothManager>()?.adapter?.bluetoothLeAdvertiser?.stopAdvertising(advertiseCallback) }
        runCatching { gatt?.close() }
        gatt = null
        onIo {
            devices.clear()
            prepared.clear()
            queue.clear()
            sending = false
            server.clear()
        }
        MacLinkState.set { it.copy(status = MacLinkStatus.NO_BLUETOOTH, macs = emptyList()) }
    }

    private fun stop() {
        closeGatt()
        if (btWatching) {
            btWatching = false
            runCatching { unregisterReceiver(btReceiver) }
        }
        hotspot?.close()
        MacLinkState.set { MacLinkLive(shizuku = it.shizuku) }
        ServiceCompat.stopForeground(this, ServiceCompat.STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onDestroy() {
        if (gatt != null || btWatching) stop()
        scope.cancel()
        io.shutdownNow()
        super.onDestroy()
    }

    private fun notification(text: String): Notification {
        val open = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP), PendingIntent.FLAG_IMMUTABLE,
        )
        val stop = PendingIntent.getService(this, 4, Intent(this, BleControlService::class.java).setAction(ACTION_STOP), PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(getString(R.string.mac_link_title))
            .setContentText(text)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(open)
            .addAction(0, getString(R.string.mac_link_stop), stop)
            .build()
    }

    companion object {
        private const val TAG = "tailbond"
        private const val CHANNEL = "mac-link"
        private const val NOTIFICATION_ID = 5
        const val ACTION_START = "dev.tailbond.app.MAC_LINK_START"
        const val ACTION_STOP = "dev.tailbond.app.MAC_LINK_STOP"

        /** The runtime permissions Bluetooth needs here (Android 12+: advertise and connect). */
        val PERMISSIONS: Array<String> =
            if (Build.VERSION.SDK_INT >= 31) arrayOf(Manifest.permission.BLUETOOTH_ADVERTISE, Manifest.permission.BLUETOOTH_CONNECT) else emptyArray()

        fun hasPermissions(ctx: Context) = PERMISSIONS.all { ContextCompat.checkSelfPermission(ctx, it) == PackageManager.PERMISSION_GRANTED }

        fun start(ctx: Context) =
            ContextCompat.startForegroundService(ctx, Intent(ctx, BleControlService::class.java).setAction(ACTION_START))

        fun stop(ctx: Context) {
            ctx.startService(Intent(ctx, BleControlService::class.java).setAction(ACTION_STOP))
        }
    }
}
