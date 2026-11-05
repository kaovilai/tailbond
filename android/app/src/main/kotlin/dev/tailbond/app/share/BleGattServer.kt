package dev.tailbond.app.share

/*
 * The GATT-server half is untether's (https://github.com/ImOstrovskiy/
 * untether, BSD-3-Clause, Copyright (c) 2026 Oleksii Ostrovskyi — see
 * TetheringShell.kt for the notice): the encrypted characteristics, the
 * per-device MTU and long-read bookkeeping, the advertising that restarts
 * on every connection change. The protocol it carries is tailbond's
 * ([BleProto]).
 */

import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothGatt
import android.bluetooth.BluetoothGattCharacteristic
import android.bluetooth.BluetoothGattCharacteristic.PERMISSION_READ_ENCRYPTED
import android.bluetooth.BluetoothGattCharacteristic.PERMISSION_WRITE_ENCRYPTED
import android.bluetooth.BluetoothGattCharacteristic.PROPERTY_NOTIFY
import android.bluetooth.BluetoothGattCharacteristic.PROPERTY_READ
import android.bluetooth.BluetoothGattCharacteristic.PROPERTY_WRITE
import android.bluetooth.BluetoothGattDescriptor
import android.bluetooth.BluetoothGattServer
import android.bluetooth.BluetoothGattServerCallback
import android.bluetooth.BluetoothGattService
import android.bluetooth.BluetoothManager
import android.bluetooth.BluetoothProfile
import android.bluetooth.le.AdvertiseCallback
import android.bluetooth.le.AdvertiseData
import android.bluetooth.le.AdvertiseSettings
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Build
import android.os.ParcelUuid
import android.util.Log
import java.util.concurrent.ConcurrentHashMap

/**
 * The phone's BLE peripheral: advertises tailbond's control service and
 * carries [BleProto] frames over it. Every characteristic needs an
 * encrypted, bonded link: the first access from an unbonded Mac triggers
 * Android's pairing (a one-time numeric comparison on both screens), and
 * after that the link — challenge, commands, the hotspot's passphrase — is
 * encrypted on the air.
 *
 * Bluetooth off, or the BLUETOOTH_ADVERTISE/CONNECT permissions not given:
 * [state] says so and the rest of the hub keeps working; the Mac's click
 * falls back to an already-joined hotspot.
 */
@SuppressLint("MissingPermission")
class BleGattServer(
    private val ctx: Context,
    private val proto: BleProto,
) {
    /** off, on, denied, unsupported — for the UI and the debug status. */
    @Volatile var state: String = "off"
        private set

    private val bt get() = ctx.getSystemService(BluetoothManager::class.java)
    private var gatt: BluetoothGattServer? = null
    private val subscribers = ConcurrentHashMap<String, BluetoothDevice>()
    private val mtu = ConcurrentHashMap<String, Int>()

    /** Long reads in flight, by device (the challenge), and the last frame announced for read-back ("-state"). */
    private val reads = ConcurrentHashMap<String, ByteArray>()

    private val btReceiver = object : BroadcastReceiver() {
        override fun onReceive(c: Context, i: Intent) {
            when (i.getIntExtra(BluetoothAdapter.EXTRA_STATE, -1)) {
                BluetoothAdapter.STATE_ON -> open()
                BluetoothAdapter.STATE_TURNING_OFF -> close()
            }
        }
    }
    private val advertiseCallback = object : AdvertiseCallback() {
        override fun onStartFailure(errorCode: Int) {
            state = "denied"
            Log.w(TAG, "ble: advertising failed ($errorCode)")
        }
    }

    init {
        proto.out = { device, frame -> notify(device, frame) }
        // a protected system broadcast, but the flag is still named on 33+
        if (Build.VERSION.SDK_INT >= 33) {
            ctx.registerReceiver(btReceiver, IntentFilter(BluetoothAdapter.ACTION_STATE_CHANGED), Context.RECEIVER_EXPORTED)
        } else {
            ctx.registerReceiver(btReceiver, IntentFilter(BluetoothAdapter.ACTION_STATE_CHANGED))
        }
    }

    /** Opens (or reopens) the GATT server and the advertising. */
    fun open() {
        if (gatt != null) return
        val adapter = bt?.adapter
        if (adapter?.isEnabled != true) {
            state = if (adapter == null) "unsupported" else "off"
            return
        }
        if (Build.VERSION.SDK_INT >= 31 && (
                ctx.checkSelfPermission(android.Manifest.permission.BLUETOOTH_ADVERTISE) != android.content.pm.PackageManager.PERMISSION_GRANTED ||
                    ctx.checkSelfPermission(android.Manifest.permission.BLUETOOTH_CONNECT) != android.content.pm.PackageManager.PERMISSION_GRANTED
                )
        ) {
            state = "denied"
            Log.i(TAG, "ble: no Bluetooth permission")
            return
        }
        val service = BluetoothGattService(Ble.SERVICE, BluetoothGattService.SERVICE_TYPE_PRIMARY).apply {
            addCharacteristic(BluetoothGattCharacteristic(Ble.CHALLENGE, PROPERTY_READ, PERMISSION_READ_ENCRYPTED))
            addCharacteristic(BluetoothGattCharacteristic(Ble.COMMAND, PROPERTY_WRITE, PERMISSION_WRITE_ENCRYPTED))
            addCharacteristic(
                BluetoothGattCharacteristic(Ble.STATE, PROPERTY_READ or PROPERTY_NOTIFY, PERMISSION_READ_ENCRYPTED).apply {
                    addDescriptor(
                        BluetoothGattDescriptor(
                            Ble.CCCD,
                            BluetoothGattDescriptor.PERMISSION_READ_ENCRYPTED or BluetoothGattDescriptor.PERMISSION_WRITE_ENCRYPTED,
                        ),
                    )
                },
            )
        }
        gatt = bt?.openGattServer(ctx, callback)?.apply { addService(service) }
        startAdvertising()
        state = if (gatt != null) "on" else "denied"
        Log.i(TAG, "ble: gatt server ${if (gatt != null) "up" else "failed"}")
    }

    fun close() {
        runCatching { bt?.adapter?.bluetoothLeAdvertiser?.stopAdvertising(advertiseCallback) }
        gatt?.close()
        gatt = null
        val was = subscribers.keys.toList()
        subscribers.clear()
        mtu.clear()
        reads.clear()
        for (d in was) proto.deviceGone(d)
        state = "off"
    }

    fun destroy() {
        close()
        runCatching { ctx.unregisterReceiver(btReceiver) }
    }

    /** Pushes the status to every connected Mac (the service's tick). */
    fun pushStatus() {
        for (d in subscribers.keys) proto.pushStatus(d)
    }

    /** Legacy connectable advertising stops on connect, so it restarts on every connection change. */
    private fun startAdvertising() {
        val advertiser = bt?.adapter?.bluetoothLeAdvertiser ?: return
        advertiser.stopAdvertising(advertiseCallback)
        advertiser.startAdvertising(
            AdvertiseSettings.Builder()
                .setAdvertiseMode(AdvertiseSettings.ADVERTISE_MODE_BALANCED)
                .setTxPowerLevel(AdvertiseSettings.ADVERTISE_TX_POWER_MEDIUM)
                .setConnectable(true)
                .setTimeout(0)
                .build(),
            AdvertiseData.Builder()
                .addServiceUuid(ParcelUuid(Ble.SERVICE))
                .setIncludeDeviceName(false)
                .build(),
            advertiseCallback,
        )
    }

    /**
     * One frame per notification, whole ([flags][frame], a single chunk):
     * Android queues notifications in order, so frames never interleave.
     * A frame that doesn't fit the ATT MTU is announced with an empty
     * value — the Mac then reads the characteristic (a long read), which
     * [callback] serves from [reads] (untether's pattern, kept).
     */
    private fun notify(device: String, frame: ByteArray) {
        val g = gatt ?: return
        val d = subscribers[device] ?: return
        val ch = g.getService(Ble.SERVICE)?.getCharacteristic(Ble.STATE) ?: return
        val mtu = mtu[device] ?: 23
        var value = byteArrayOf(BleProto.CHUNK_WHOLE) + frame // one chunk, flags and all
        if (value.size > mtu - 3) {
            reads["$device-state"] = value
            value = ByteArray(0) // too long to push: the Mac reads it back
        }
        if (Build.VERSION.SDK_INT >= 33) {
            g.notifyCharacteristicChanged(d, ch, false, value)
        } else {
            @Suppress("DEPRECATION")
            ch.value = value
            @Suppress("DEPRECATION")
            g.notifyCharacteristicChanged(d, ch, false)
        }
    }

    private val callback = object : BluetoothGattServerCallback() {
        override fun onConnectionStateChange(device: BluetoothDevice, status: Int, newState: Int) {
            if (newState == BluetoothProfile.STATE_CONNECTED) {
                Log.i(TAG, "ble: ${device.address} connected")
            } else {
                Log.i(TAG, "ble: ${device.address} disconnected")
                subscribers.remove(device.address)
                mtu.remove(device.address)
                reads.keys.removeAll { it == device.address || it.startsWith("${device.address}-") }
                proto.deviceGone(device.address)
            }
            startAdvertising()
        }

        override fun onMtuChanged(device: BluetoothDevice, mtu: Int) {
            this@BleGattServer.mtu[device.address] = mtu
        }

        override fun onCharacteristicReadRequest(device: BluetoothDevice, requestId: Int, offset: Int, ch: BluetoothGattCharacteristic) {
            when (ch.uuid) {
                // a read at offset 0 starts a (possibly long) read: the
                // whole value comes from one snapshot, so the blobs of a
                // read belong together (a fresh challenge per blob would
                // splice two nonces into one)
                Ble.CHALLENGE -> serveRead(device, requestId, offset) { proto.challenge(device.address) }
                Ble.STATE -> serveRead(device, requestId, offset) { reads["${device.address}-state"] ?: byteArrayOf(BleProto.CHUNK_WHOLE, 0, 0) }
                else -> gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_READ_NOT_PERMITTED, 0, null)
            }
        }

        private fun serveRead(device: BluetoothDevice, requestId: Int, offset: Int, value: () -> ByteArray) {
            val v = if (offset == 0) value().also { reads[device.address] = it } else reads[device.address] ?: value()
            if (offset > v.size) {
                gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_INVALID_OFFSET, 0, null)
            } else {
                gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, v.copyOfRange(offset, v.size))
            }
        }

        override fun onCharacteristicWriteRequest(
            device: BluetoothDevice, requestId: Int, ch: BluetoothGattCharacteristic,
            preparedWrite: Boolean, responseNeeded: Boolean, offset: Int, value: ByteArray,
        ) {
            // tailbond's frames fit one write at any MTU (the Mac chunks
            // them itself); prepared writes are assembled the same way
            var status = BluetoothGatt.GATT_SUCCESS
            if (ch.uuid == Ble.COMMAND) {
                proto.onWrite(device.address, value)
            } else {
                status = BluetoothGatt.GATT_WRITE_NOT_PERMITTED
            }
            if (responseNeeded) gatt?.sendResponse(device, requestId, status, offset, value)
        }

        override fun onDescriptorReadRequest(device: BluetoothDevice, requestId: Int, offset: Int, d: BluetoothGattDescriptor) {
            val on = subscribers.containsKey(device.address)
            val value = if (on) BluetoothGattDescriptor.ENABLE_NOTIFICATION_VALUE else BluetoothGattDescriptor.DISABLE_NOTIFICATION_VALUE
            gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, 0, value)
        }

        override fun onDescriptorWriteRequest(
            device: BluetoothDevice, requestId: Int, d: BluetoothGattDescriptor,
            preparedWrite: Boolean, responseNeeded: Boolean, offset: Int, value: ByteArray,
        ) {
            if (value.contentEquals(BluetoothGattDescriptor.ENABLE_NOTIFICATION_VALUE)) subscribers[device.address] = device
            else subscribers.remove(device.address)
            if (responseNeeded) gatt?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, 0, null)
        }
    }

    private companion object {
        const val TAG = "tailbond"
    }
}
