from dataclasses import dataclass
from marshmallow_dataclass import class_schema


@dataclass
class StreamStats:
    """Dataclass representing streaming statistics."""

    target_bitrate_kbps: float
    avg_bitrate_kbps: float
    peak_bitrate_kbps: float
    duration_sec: int
    dropped_frames: int = 0

    @property
    def quality(self) -> int:

        return (self.avg_bitrate_kbps/self.target_bitrate_kbps) * 100

    def __str__(self):
        # Using self's attributes to construct the output string
        return (
            '\n'
            + '=' * 60
            + '\n📊 BANDWIDTH TEST RESULTS\n'
            + '=' * 60
            + f'\nTarget Bitrate:    {self.target_bitrate_kbps} kbps'  # No target_bitrate_kbps given in constructor
            f'\nActual Avg Bitrate:  {self.avg_bitrate_kbps:.1f} kbps'
            f'\nPeak Bitrate:        {self.peak_bitrate_kbps:.1f} kbps'
            f'\nDropped Frames:      {self.dropped_frames}'
            f'\nDuration:            {self.duration_sec:.1f} seconds'
        )


@dataclass
class IngestServer:
    _id: int
    availability: float
    default: bool
    name: str
    priority: int
    url_template: str
    url_template_secure: str

    @property
    def region(self) -> str | None:
        """Возвращает регион из имени сервера"""
        if self.name == 'Default':
            return 'Default'
        parts = self.name.split(':')
        return parts[0].strip() if len(parts) > 1 else None


@dataclass
class ResponseModel:
    ingests: list['IngestServer']


ResponseSchema = class_schema(ResponseModel)
StreamStatsSchema = class_schema(StreamStats)
