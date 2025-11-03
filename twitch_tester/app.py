from twitch_tester.bin import test_twitch_bandwidth
from twitch_tester.models import IngestServer
from twitch_tester.lib import rtmps_ping

from typing import Iterable
from urllib.parse import urlparse


'''
У нас есть список серверов
В первый проход мы получаем пинг до них

Во сторой проход скорость загрузки
'''


class App:
    def __init__(
        self,
        stream_key,
        duration: int = 10,
        target_bitrate_kbps: int = 10_000,
    ):
        self.ping_results = {}
        self.ffmpeg_results = {}
        self.stream_key = stream_key
        self.duration = duration
        self.target_bitrate_kbps = target_bitrate_kbps
        self.map_servers = {}

    def test_servers(self, servers: Iterable[IngestServer]):
        self.ping_results = {server.name: self.test_ping(server) for server in servers}
        for server in servers:
            self.map_servers[server.name] = server
            self.ffmpeg_results[server.name] = self.test_binupload(server)
        for server in self.ffmpeg_results.keys():
            ping_result = self.ping_results[server]
            if ping_result.get('success'):
                self.ffmpeg_results[server].update_ping(ping_result['total_rtt_sec'])

        sorted_results = sorted(self.ffmpeg_results.items(), key=lambda kv: kv[1].quality, reverse=True)
        for loc, stat in sorted_results:
            rtt = int(self.ping_results.get(loc, {}).get('total_rtt_sec', 0) * 1000)
            print(f'\n{self.map_servers[loc].url_template}')
            print(f'{loc} - {stat.quality:.2f} {stat.ping} avgb: {stat.avg_bitrate_kbps} kbps')

    def test_ping(self, server: IngestServer):
        host = self._extract_host(server.url_template)
        print(f'testing {server.name} ...', end='')
        ping_result = rtmps_ping(host)
        print(ping_result.get('total_rtt_sec', 0) * 1000)
        return ping_result

    def test_binupload(self, server: IngestServer):
        print(f'Starting {server.name} {server.url_template} {self.duration}s bandwidth test at {self.target_bitrate_kbps} kbps... ', end='')
        url = f'{server.url_template.format(stream_key=self.stream_key)}?bandwidthtest'
        result = test_twitch_bandwidth(url, self.stream_key, duration=self.duration)
        if not result:
            print('error')
            return
        print(f'avg: {result.avg_bitrate_kbps:.2f}kbps')
        return result

    @staticmethod
    def _extract_host(rtmp_url):
        """
        Extracts the hostname from an RTMP URL.

        :param rtmp_url: An RTMP URL (e.g., rtmp://mnl01.contribute.live-video.net/app/{stream_key})
        :return: Hostname (string)
        """
        parsed_url = urlparse(rtmp_url)
        return parsed_url.hostname
