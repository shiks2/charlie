#!/usr/bin/env python3

import asyncio
import json
from datetime import datetime, timezone

import websockets
from websockets.exceptions import InvalidStatus

CHARGER_ID = "CHARGER-SIM-001"
WS_URL = f"ws://localhost:8887/ocpp/{CHARGER_ID}"
FALLBACK_WS_URL = f"ws://localhost:8887/{CHARGER_ID}"
SUBPROTOCOL = "ocpp1.6j"


def iso_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


async def send_and_wait(ws, msg: list) -> list:
    outbound = json.dumps(msg, separators=(",", ":"))
    print(f"SEND: {outbound}")
    await ws.send(outbound)

    inbound = await ws.recv()
    print(f"RECV: {inbound}")

    parsed = json.loads(inbound)
    if not isinstance(parsed, list) or len(parsed) < 3 or parsed[0] != 3:
        raise RuntimeError(f"Unexpected OCPP response format: {parsed}")

    if parsed[1] != msg[1]:
        raise RuntimeError(f"Response unique-id mismatch: expected {msg[1]}, got {parsed[1]}")

    return parsed


async def main() -> None:
    ws = None
    active_url = WS_URL
    print(f"Connecting to {WS_URL} with subprotocol={SUBPROTOCOL}")

    try:
        ws = await websockets.connect(WS_URL, subprotocols=[SUBPROTOCOL])
    except InvalidStatus as err:
        if err.response is not None and err.response.status_code == 404:
            print(
                f"Primary URL returned 404. Retrying with fallback URL: {FALLBACK_WS_URL}"
            )
            active_url = FALLBACK_WS_URL
            ws = await websockets.connect(FALLBACK_WS_URL, subprotocols=[SUBPROTOCOL])
        else:
            raise

    async with ws:
        print(f"Connected on {active_url}")
        # 1) BootNotification
        boot = [
            2,
            "msg-001",
            "BootNotification",
            {
                "chargePointVendor": "Kazam",
                "chargePointModel": "EVC-7K",
                "chargePointSerialNumber": "KZM-2024-001",
                "firmwareVersion": "1.0.3",
            },
        ]
        await send_and_wait(ws, boot)

        # 2) Sleep 1 second
        await asyncio.sleep(1)

        # 3) StatusNotification (connector 1, Available)
        status = [
            2,
            "msg-002",
            "StatusNotification",
            {
                "connectorId": 1,
                "errorCode": "NoError",
                "status": "Available",
                "timestamp": iso_now(),
            },
        ]
        await send_and_wait(ws, status)

        # 4) Sleep 1 second
        await asyncio.sleep(1)

        # 5) Heartbeat
        heartbeat = [2, "msg-003", "Heartbeat", {}]
        await send_and_wait(ws, heartbeat)

        # 6) Sleep 1 second
        await asyncio.sleep(1)

        # 7) StartTransaction
        start_txn = [
            2,
            "msg-004",
            "StartTransaction",
            {
                "connectorId": 1,
                "idTag": "RFID-TEST-001",
                "meterStart": 0,
                "timestamp": iso_now(),
            },
        ]
        start_resp = await send_and_wait(ws, start_txn)
        start_payload = start_resp[2]
        if not isinstance(start_payload, dict) or "transactionId" not in start_payload:
            raise RuntimeError(f"StartTransaction response missing transactionId: {start_resp}")
        transaction_id = start_payload["transactionId"]
        print(f"Extracted transactionId={transaction_id}")

        # 8) Sleep 2 seconds
        await asyncio.sleep(2)

        # 9) MeterValues
        meter_values = [
            2,
            "msg-005",
            "MeterValues",
            {
                "connectorId": 1,
                "transactionId": transaction_id,
                "meterValue": [
                    {
                        "timestamp": iso_now(),
                        "sampledValue": [
                            {
                                "value": "5200",
                                "context": "Sample.Periodic",
                                "format": "Raw",
                                "measurand": "Energy.Active.Import.Register",
                                "unit": "Wh",
                            }
                        ],
                    }
                ],
            },
        ]
        await send_and_wait(ws, meter_values)

        # 10) Sleep 2 seconds
        await asyncio.sleep(2)

        # 11) StopTransaction
        stop_txn = [
            2,
            "msg-006",
            "StopTransaction",
            {
                "transactionId": transaction_id,
                "meterStop": 14200,
                "timestamp": iso_now(),
                "reason": "Local",
            },
        ]
        await send_and_wait(ws, stop_txn)

        # 12) Sleep 1 second then close connection
        await asyncio.sleep(1)

    print("Connection closed")


if __name__ == "__main__":
    asyncio.run(main())
