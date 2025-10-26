import argparse
import os
import sys

from twitch_tester.app import App
from twitch_tester.server import TwitchTest


def main():
    parser = argparse.ArgumentParser(description='CLI App for Managing Servers')

    # Adding mutually exclusive group for clarity
    group = parser.add_mutually_exclusive_group()
    group.add_argument('--servers', action='store_true', help='List all available servers')
    group.add_argument('--regions', action='store_true', help='List all available regions')
    group.add_argument(
        '--test_region',
        dest='filter_server_by_region',
        nargs='?',
        const='',
        metavar='REGION',
        help='Filter servers by region. Specify REGION after "--server" argument.',
    )

    args = parser.parse_args()
    stream_key = os.environ.get('STREAM_KEY', None)
    if not stream_key:
        print('SET STREAM_KEY env')
        sys.exit(1)

    t = TwitchTest()
    t.request_servers()
    app = App(stream_key=stream_key)

    if args.servers:
        print('Available Servers:')
        for server in t.get_servers():
            print(f'{server.name} {server.url_template}')
    elif args.regions:
        REGIONS = t.get_regions()
        print('Available Regions:', ', '.join(REGIONS))
    elif args.filter_server_by_region:
        filtered_servers = [s for s in t.get_servers() if s.region.lower() == args.filter_server_by_region.lower()]
        if len(filtered_servers) > 0:
            print(f"Servers in Region '{args.filter_server_by_region}'")
            app.test_servers(filtered_servers)
        else:
            print(f"No servers found in region '{args.filter_server_by_region}'.")
    else:
        parser.print_help()


if __name__ == '__main__':
    main()
