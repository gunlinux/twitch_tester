import typing
import requests
from twitch_tester.models import ResponseSchema, ResponseModel, IngestServer
from marshmallow import ValidationError


class TwitchTest:
    def __init__(self) -> None:
        self.regions: set[str | None] | None = None
        self.data: ResponseModel | None = None

    def request_servers(self) -> None:
        url = 'https://ingest.twitch.tv/ingests'  # No auth needed
        response = requests.get(url)

        if response.status_code == 200:
            self.server_data(response.json())
        else:
            print(f'HTTP error: {response.status_code}')

    def server_data(self, data: dict) -> None:
        try:
            self.data = typing.cast(ResponseModel, ResponseSchema().load(data))
            self.regions = self.get_regions()
        except ValidationError as e:
            print('Validation failed:', e.messages)
            self.data = None
            self.regions = None

    def get_regions(self) -> set[str | None] | None:
        if self.data is None:
            return None
        return {server.region for server in self.data.ingests}

    def get_servers(self) -> list[IngestServer]:
        if self.data is None:
            return []
        return [server for server in self.data.ingests]
