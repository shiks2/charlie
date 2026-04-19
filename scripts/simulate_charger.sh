#!/usr/bin/env bash

set -euo pipefail

if ! command -v wscat >/dev/null 2>&1; then
	echo "wscat not found. Install it with: npm install -g wscat"
	exit 1
fi

WS_URL="ws://localhost:8887/CHARGER-SIM-001"
SUBPROTOCOL="ocpp1.6j"

{
	# 1. BootNotification
	echo '[2,"msg-001","BootNotification",{"chargePointVendor":"Kazam","chargePointModel":"EVC-7K","chargePointSerialNumber":"KZM-2024-001","firmwareVersion":"1.0.3"}]'
	sleep 2

	# 2. StatusNotification (connector 1, Available)
	echo '[2,"msg-002","StatusNotification",{"connectorId":1,"errorCode":"NoError","status":"Available"}]'
	sleep 2

	# 3. Heartbeat
	echo '[2,"msg-003","Heartbeat",{}]'
	sleep 2

	# 4. StartTransaction
	echo '[2,"msg-004","StartTransaction",{"connectorId":1,"idTag":"RFID-TEST-001","meterStart":0,"timestamp":"2026-04-18T17:00:00Z"}]'
	sleep 2

	# 5. MeterValues (mid-session reading)
	echo '[2,"msg-005","MeterValues",{"connectorId":1,"transactionId":1,"meterValue":[{"timestamp":"2026-04-18T17:05:00Z","sampledValue":[{"value":"5200","context":"Sample.Periodic","format":"Raw","measurand":"Energy.Active.Import.Register","unit":"Wh"}]}]}]'
	sleep 2

	# 6. StopTransaction
	echo '[2,"msg-006","StopTransaction",{"transactionId":1,"meterStop":14200,"timestamp":"2026-04-18T17:30:00Z","reason":"Local"}]'
} | wscat -c "$WS_URL" -s "$SUBPROTOCOL"
