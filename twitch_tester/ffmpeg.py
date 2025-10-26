import subprocess
import re

from twitch_tester.models import StreamStats


class FFMpegMissing(Exception):
    pass


class TimeoutExpired(Exception):
    pass


def test_twitch_bandwidth(
    rtmp_endpoint: str, duration: int = 30, target_bitrate_kbps: int = 1000
) -> StreamStats | None:
    cmd = [
        'ffmpeg',
        '-f',
        'lavfi',
        '-i',
        f'testsrc=size=1280x720:rate=30:duration={duration}',
        '-f',
        'lavfi',
        '-i',
        f'sine=frequency=1000:duration={duration}',
        '-c:v',
        'libx264',
        '-preset',
        'ultrafast',
        '-tune',
        'zerolatency',
        '-b:v',
        f'{target_bitrate_kbps}k',
        '-maxrate',
        f'{target_bitrate_kbps}k',
        '-bufsize',
        f'{2 * target_bitrate_kbps}k',
        '-minrate',
        f'{target_bitrate_kbps}k',
        '-pix_fmt',
        'yuv420p',
        '-c:a',
        'aac',
        '-b:a',
        '128k',
        '-f',
        'flv',
        '-flush_packets',
        '0',
        '-fflags',
        'nobuffer',
        '-flags',
        'low_delay',
        '-progress',
        'pipe:2',
        rtmp_endpoint,
    ]

    # Initialize tracking variables
    total_bits = 0
    last_time_seconds = 0.0
    max_bitrate_kbps = 0.0
    progress_pattern = re.compile(r'(.+?)=(.+)')
    dropped_frames = 0
    progress_data = {}

    try:
        # Start the subprocess
        proc = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1, universal_newlines=True
        )

        # Read stderr line by line
        bitrates = []
        if proc.stderr:
            for line in iter(proc.stderr.readline, ''):
                # Parse progress lines (from -progress)
                print(line)
                if '=' in line and not line.startswith('['):
                    match = progress_pattern.match(line.strip())
                    if match:
                        key, value = match.groups()
                        progress_data[key] = value

                        if key == 'bitrate':
                            try:
                                br = float(value.split('k')[0])
                                max_bitrate_kbps = max(max_bitrate_kbps, br)
                                bitrates.append(br)
                            except (ValueError, IndexError):
                                pass

                        if key == 'total_size':
                            try:
                                total_bits = int(value) * 8  # bytes → bits
                            except ValueError:
                                pass

                        if key == 'out_time_ms':
                            try:
                                last_time_seconds = int(value) / 1_000_000.0  # microseconds → seconds
                            except ValueError:
                                pass
                        if key == 'dropped_frames':
                            try:
                                dropped_frames += int(value)
                            except ValueError:
                                pass

        # Wait for process completion
        proc.wait(timeout=5)

        # Calculate average bitrate
        avg_bitrate_kbps = 0.0
        if last_time_seconds > 1:
            avg_bitrate_kbps = sum(bitrates) / len(bitrates)

        # Determine success based on criteria
        return StreamStats(
            target_bitrate_kbps=target_bitrate_kbps,
            avg_bitrate_kbps=avg_bitrate_kbps,
            peak_bitrate_kbps=max_bitrate_kbps,
            duration_sec=last_time_seconds,
            dropped_frames=dropped_frames,
        )

    except subprocess.TimeoutExpired:
        proc.kill()
        raise TimeoutExpired
    except FileNotFoundError:
        raise FFMpegMissing
