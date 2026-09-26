#!/usr/bin/env python3
"""Minimal RFC6455 WebSocket listener (stdlib only) for SRAT /ws testing.

The SRAT /ws endpoint speaks real WebSocket (RFC 6455) with SSE-style text
payloads (``id:``/``event:``/``data:`` lines inside text frames) and requires
an ``Origin`` header — plain ``curl -N`` returns ``400 Bad Request``.

Usage:
    python3 ws_listen.py <logfile> [seconds] [url] [origin]

Appends one ``[timestamp] <payload>`` line per complete text message,
responds to ping with pong, and logs ``# CLOSE``/``# EOF`` markers.
Exit 0 when frames arrived, 2 when the stream stayed silent, 1 on error.

Proven pattern for docs/test/backend/006.00x WS cases.
"""

import base64
import os
import socket
import ssl
import struct
import sys
import time
from urllib.parse import urlparse


def ws_connect(url, origin):
    u = urlparse(url)
    host = u.hostname
    port = u.port or (443 if u.scheme == "wss" else 80)
    key = base64.b64encode(os.urandom(16)).decode()
    sock = socket.create_connection((host, port), timeout=10)
    if u.scheme == "wss":
        sock = ssl.create_default_context().wrap_socket(sock, server_hostname=host)
    req = (
        f"GET {u.path or '/'} HTTP/1.1\r\n"
        f"Host: {host}:{port}\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {key}\r\n"
        "Sec-WebSocket-Version: 13\r\n"
        f"Origin: {origin}\r\n"
        "\r\n"
    )
    sock.sendall(req.encode())
    resp = b""
    while b"\r\n\r\n" not in resp:
        chunk = sock.recv(4096)
        if not chunk:
            raise RuntimeError("handshake EOF: " + resp.decode("latin1"))
        resp += chunk
    status = resp.decode("latin1").split("\r\n", 1)[0]
    if "101" not in status:
        raise RuntimeError("handshake failed: " + status)
    return sock, resp.split(b"\r\n\r\n", 1)[1]


def read_frames(sock, pending, logf, deadline):
    buf = bytearray(pending)
    msg = bytearray()
    frames = 0
    sock.settimeout(1.0)
    while time.time() < deadline:
        try:
            data = sock.recv(65536)
        except TimeoutError:
            continue
        if not data:
            logf.write(f"# EOF {time.time():.0f}\n")
            logf.flush()
            break
        buf += data
        while len(buf) >= 2:
            b1, b2 = buf[0], buf[1]
            opcode = b1 & 0x0F
            masked = b2 & 0x80
            length = b2 & 0x7F
            idx = 2
            if length == 126:
                if len(buf) < 4:
                    break
                length = struct.unpack("!H", buf[2:4])[0]
                idx = 4
            elif length == 127:
                if len(buf) < 10:
                    break
                length = struct.unpack("!Q", buf[2:10])[0]
                idx = 10
            if masked:
                if len(buf) < idx + 4:
                    break
                mask = bytes(buf[idx : idx + 4])
                idx += 4
            if len(buf) < idx + length:
                break
            payload = bytearray(buf[idx : idx + length])
            del buf[: idx + length]
            if masked:
                for i in range(len(payload)):
                    payload[i] ^= mask[i % 4]
            if opcode == 0x8:  # close
                code = struct.unpack("!H", payload[:2])[0] if len(payload) >= 2 else -1
                reason = (
                    payload[2:].decode("utf-8", "replace") if len(payload) > 2 else ""
                )
                logf.write(f"# CLOSE {time.time():.0f} code={code} reason={reason!r}\n")
                logf.flush()
                return frames
            if opcode == 0x9:  # ping -> pong (client frames MUST be masked)
                mask = os.urandom(4)
                masked = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
                sock.sendall(bytes([0x8A, 0x80 | len(payload)]) + mask + masked)
                continue
            if opcode == 0xA:  # pong
                continue
            if opcode in (0x0, 0x1):  # continuation / text
                msg += payload
                if b1 & 0x80:
                    ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
                    logf.write(f"[{ts}] {msg.decode('utf-8', 'replace')}\n")
                    logf.flush()
                    frames += 1
                    msg = bytearray()
    return frames


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    logfile = sys.argv[1]
    seconds = int(sys.argv[2]) if len(sys.argv) > 2 else 120
    url = sys.argv[3] if len(sys.argv) > 3 else "ws://192.168.0.68:3000/ws"
    origin = sys.argv[4] if len(sys.argv) > 4 else "http://localhost:3080"
    try:
        sock, pending = ws_connect(url, origin)
    except Exception as e:  # noqa: BLE001
        print(f"CONNECT FAILED: {e}")
        return 1
    print(f"CONNECTED {url} origin={origin} logging to {logfile} for {seconds}s")
    with open(logfile, "a") as logf:
        logf.write(f"# CONNECT {time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())}\n")
        logf.flush()
        frames = read_frames(sock, pending, logf, time.time() + seconds)
    print(f"DONE frames={frames}")
    return 0 if frames > 0 else 2


if __name__ == "__main__":
    sys.exit(main())
