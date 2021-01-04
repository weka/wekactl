#!/usr/bin/env python


import http.client
import json
import os
import re
import socket
import sys
import uuid
from operator import itemgetter
from urllib.parse import urlparse

httpclient = http.client

DEFAULT_SCHEME = 'http'
DEFAULT_HOST = os.environ.get('WEKA_HOST', None)
DEFAULT_PORT = 14000
DEFAULT_PATH = '/api/v1'
# noinspection SpellCheckingInspection
ALPHABET = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

DEFAULT_CONNECTION_TIMEOUT = 10


def parse_url(url, default_port=None, default_path='/'):
    parsed_url = urlparse(url)
    scheme = parsed_url.scheme if parsed_url.scheme else DEFAULT_SCHEME
    if scheme not in ['http', 'https']:
        scheme = DEFAULT_SCHEME
    if default_port is None:
        default_port = 443 if scheme == 'https' else 80
    m = re.match(r'^(?:(?:http|https)://)?(.+?)(?::(\d+))?(/.*)?$', str(url), re.I)
    assert m
    return scheme, m.group(1), m.group(2) or default_port, m.group(3) or default_path


def get_scheme_host_port_and_path(host):
    return parse_url(host, DEFAULT_PORT, DEFAULT_PATH)


class HttpException(Exception):
    def __init__(self, error_code, error_msg):
        self.error_code = error_code
        self.error_msg = error_msg


class JsonRpcException(Exception):
    def __init__(self, json_error):
        self.orig_json = json_error
        self.code = json_error['code']
        self.message = json_error['message']
        self.data = json_error.get('data', None)


class WapiException(Exception):
    def __init__(self, message):
        self.message = message


def _get_credentials_from_environment():
    org = os.environ['WEKA_ORG'] if ('WEKA_ORG' in os.environ) else None
    for username_var, password_var in (('WEKA_USERNAME', 'WEKA_PASSWORD'), ('WEKA_USER', 'WEKA_PASS')):
        if (username_var in os.environ) and (password_var in os.environ):
            return org, os.environ[username_var], os.environ[password_var]
    return None


class WekaManagementCredentials:
    CREDENTIALS_FILENAME = '~/.weka/cli.conf'
    DEFAULT_CREDENTIALS = (None, 'admin', 'admin')

    def __init__(self):
        creds = _get_credentials_from_environment()
        self.org, self.username, self.password = self.DEFAULT_CREDENTIALS if creds is None else creds
        self.authorization = None

    def login(self, host):
        try:
            anon_conn = JsonRpcConnection(DEFAULT_SCHEME, host, DEFAULT_PORT, DEFAULT_PATH)
            params = dict(username=self.username, password=self.password)
            if self.org is not None:
                params.update(org=self.org)
            authorization = anon_conn.rpc('user_login', params, DEFAULT_PATH, authenticate=False)
        except HttpException as error:
            if error.error_code == httpclient.UNAUTHORIZED:
                print('error: Incorrect username or password.')
                print()
                print('If you pass your credentials using environment variables, please make')
                print('sure to pass them in WEKA_USERNAME/WEKA_PASSWORD environment variables.')
                print()
                print('You may need to pass WEKA_ORG if you are not in the root organization.')
                print()
                print('If you keep your credentials in {0}'.format(self.CREDENTIALS_FILENAME))
                print('please make sure to update the file and try again.')
                print()
                raise SystemExit(1)
            raise
        self.authorization = '{0} {1}'.format(authorization['token_type'], authorization['access_token'])

    def get_auth_headers(self):
        headers = {}
        if self.authorization:
            headers["Authorization"] = self.authorization
        return headers


class JsonRpcConnection:
    def __init__(self, scheme, host, port, path, timeout=DEFAULT_CONNECTION_TIMEOUT):

        self._host = host
        self._port = port
        self._conn = httpclient.HTTPConnection(host=host, port=port,
                                               timeout=timeout) if scheme == 'http' else httpclient.HTTPSConnection(
            host=host, port=port, timeout=timeout)
        self._path = path
        self._timeout = timeout
        self._creds = WekaManagementCredentials()
        self.headers = self._creds.get_auth_headers()

    @staticmethod
    def format_request(message_id, method, params):
        return dict(jsonrpc='2.0',
                    method=method,
                    params=params,
                    id=message_id)

    @staticmethod
    def unique_id(alphabet=ALPHABET):
        number = uuid.uuid4().int
        result = ''
        while number != 0:
            number, i = divmod(number, len(alphabet))
            result = alphabet[i] + result
        return result

    def rpc_with_header_getter(self, method, params=None, path=None, authenticate=True):
        message_id = self.unique_id()
        request = self.format_request(message_id, method, params)

        response = None
        response_body = None
        for i in range(2):
            self._conn.request('POST', self._path if not path else path, json.dumps(request), self.headers)
            response = self._conn.getresponse()
            response_body = response.read().decode('utf-8')

            if authenticate and response.status == httpclient.UNAUTHORIZED:
                self._creds.login(self._host)
                self.headers = self._creds.get_auth_headers()
                continue
            if response.status in (httpclient.OK, httpclient.CREATED, httpclient.ACCEPTED):
                response_object = json.loads(response_body)
                if 'error' in response_object:
                    raise JsonRpcException(response_object['error'])
                return response_object['result'], response.getheader
            if response.status == httpclient.MOVED_PERMANENTLY:
                scheme, host, port, self._path = parse_url(response.getheader('Location'))
                if scheme == 'http':
                    self._conn = httpclient.HTTPConnection(host=host, port=port, timeout=self._conn.timeout)
                else:
                    httpclient.HTTPSConnection(host=host, port=port, timeout=self._conn.timeout)
            else:
                raise HttpException(response.status, response.reason)

        assert response is not None
        assert response_body is not None
        raise HttpException(response.status, response_body)

    def rpc(self, *args, **kwargs):
        return self.rpc_with_header_getter(*args, **kwargs)[0]


class ApiParseException(Exception):
    pass


class MissingArgumentValue(ApiParseException):
    def __init__(self, param_name):
        super(MissingArgumentValue, self).__init__("Missing argument value for param '%s'" % param_name)


def print_results(results):
    return results


def wapi_main(host, method, named_args):
    host = host  # TODO: get host
    method_name = method.replace('-', '_')

    host_port_and_path = get_scheme_host_port_and_path(host)

    if host_port_and_path:
        scheme, host, port, path = host_port_and_path
        con = JsonRpcConnection(scheme, host, port, path)
    else:
        con = None

    if con is None:
        if method_name:
            print('Could not connect to host')
            return 1
        else:
            print("No connection required, no host")
            return

    con.headers['Client-Type'] = 'CLI'

    try:
        spec = con.rpc('getServiceSpec', {'method': method_name})
        print(spec)
        rpc_result, header_getter = con.rpc_with_header_getter(method_name, named_args)
        return print_results(rpc_result)

    except socket.timeout as e:
        print('Timed out on connection', e)
        return 1
    except IOError as e:
        print(e)
        return e.errno
    except HttpException as e:
        print(e)
        return 1
    except JsonRpcException as e:
        if e.code in [-32601]:
            print("Unknown method")
        else:
            error = []
            if e.data and isinstance(e.data, dict) and e.data.get('exceptionClass', None):
                e.data.pop('exceptionText', None)
                exception_class = e.data.pop('exceptionClass')
                if isinstance(exception_class, list):
                    error.append('Error: %s' % (e.message,))
                else:
                    error.append('[%s] %s' % (exception_class, e.message))
            else:
                error.append('%s (%s): ' % (e.message, e.code))
            if e.data:
                if isinstance(e.data, dict):
                    error.append(print_results(dict((k, v) for k, v in e.data.items() if str(v) not in e.message)))
                else:
                    error.append(print_results(e.data))
            if e.code in [-32602]:
                print(e, method_name)
            else:
                print(error, file=sys.stderr)
        return 1


def find_hosts_with_inactive_drives(deactivating_hosts, host_to_drive_and_status):
    """
    Check if each host has all inactive drives. If so, add it to a list of hosts to deactivate
    if the drives are active, add them to the active hosts+drives list
    """

    hosts_with_inactive_drives = []
    fully_active_hosts_and_drives = {}
    for host, statuses_and_drives in host_to_drive_and_status.items():
        deactivating = 0
        inactive = 0
        active = 0
        if host in deactivating_hosts:
            continue
        for drive in statuses_and_drives:
            status = drive.keys()[0]
            if status == "ACTIVE" or status == "PHASING_IN":
                if host not in fully_active_hosts_and_drives:
                    fully_active_hosts_and_drives[host] = []
                fully_active_hosts_and_drives[host].append(drive[status])
                active += 1
            elif status == "PHASING_OUT":
                deactivating += 1
            elif status == "INACTIVE":
                inactive += 1

        if active == 0 and deactivating == 0:
            hosts_with_inactive_drives.append(host)
    return hosts_with_inactive_drives, fully_active_hosts_and_drives


def organize_hosts_data(all_hosts):
    # organize all hosts into [host_id: {instance_id, status, added_time},...] by creating a new dict
    organized_hosts = {}
    for host, data in all_hosts.iteritems():
        instance_id = data['aws']['instance_id'] if data['aws'] is not None else None
        organized_hosts[host] = {'instance_id': instance_id, 'status': data['status'], 'added_time': data['added_time']}
    return organized_hosts


def scale(ip, username, password, desired_capacity):
    # return host_list (host_id, instance_id, status (active, deactivating, inactive))
    # create requirements.txt

    os.environ["WEKA_USERNAME"] = username
    os.environ["WEKA_PASSWORD"] = password
    host_to_drive_and_status = {}
    inactive_hosts = []
    deactivating_hosts = []
    active_hosts = []

    # organize hosts by active, inactive, and deactivating
    all_hosts = wapi_main(ip, 'hosts-list', {})
    all_hosts_list = []
    for host, host_data in all_hosts.iteritems():
        host_data['host_id'] = host
        all_hosts_list.append(host_data)
        if host_data['state'] == 'INACTIVE':
            inactive_hosts.append(host)
        elif host_data['state'] == 'DEACTIVATING':
            deactivating_hosts.append(host)
        else:
            host_data["host_id"] = host
            active_hosts.append(host_data)

    # sort by date so that we get the oldest instances first
    active_hosts.sort(key=itemgetter('added_time'))

    # list all drives and check which ones are inactive
    drive_list = wapi_main(ip, 'disks-list', {'show_removed': False})
    for drive, drive_data in drive_list.iteritems():
        if drive_data['host_id'] not in host_to_drive_and_status.keys():
            host_to_drive_and_status[drive_data['host_id']] = []
        host_to_drive_and_status[drive_data['host_id']].append({drive_data['status']: drive_data['uuid']})

    hosts_with_inactive_drives, \
        fully_active_hosts_and_drives = find_hosts_with_inactive_drives(deactivating_hosts, host_to_drive_and_status)

    # deactivate hosts whose drives are all INACTIVE by sending their IDs
    wapi_main(ip, 'cluster-deactivate-hosts',
              {"host_ids": [host_id.split("<")[1].split(">")[0] for host_id in hosts_with_inactive_drives],
               "no_wait": False, "skip_resource_validation": False})

    # Check if we need to deactivate more drives
    if len(fully_active_hosts_and_drives) > desired_capacity:
        number_of_hosts_to_deactivate = len(fully_active_hosts_and_drives) - desired_capacity
        i = 0
        for host in active_hosts:
            if host['host_id'] in fully_active_hosts_and_drives:
                if i < number_of_hosts_to_deactivate:
                    wapi_main(ip, 'cluster-deactivate-drives',
                              {'drive_uuids': fully_active_hosts_and_drives[host['host_id']]})
                    i += 1
                else:
                    break

    # return updated hosts_list and instance_ids of inactive hosts
    return organize_hosts_data(all_hosts), inactive_hosts


# noinspection PyUnusedLocal
def lambda_handler(event, context):
    hosts_data, inactive = scale(event['hostname'], event['username'], event['password'], event['desired_capacity'])
    return {
        'hosts': hosts_data,
        'inactive': inactive,
    }
