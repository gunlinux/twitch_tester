import socket
import ssl
import time


def rtmps_ping(host, port=443, timeout=5):
    start = time.perf_counter()
    try:
        # Step 1: Create a TCP socket
        sock = socket.create_connection((host, port), timeout=timeout)
        connect_time = time.perf_counter()

        # Step 2: Wrap with SSL for RTMPS
        context = ssl.create_default_context()
        with context.wrap_socket(sock, server_hostname=host):
            tls_time = time.perf_counter()

        total_time = tls_time - start
        tcp_time = connect_time - start
        tls_handshake_time = tls_time - connect_time

        return {
            'success': True,
            'total_rtt_sec': total_time,
            'tcp_connect_sec': tcp_time,
            'tls_handshake_sec': tls_handshake_time,
        }
    except Exception as e:
        return {
            'success': False,
            'error': str(e),
        }
