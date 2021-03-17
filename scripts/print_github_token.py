import os
import time

import jwt
import requests
from cryptography.hazmat.backends import default_backend


def print_github_token():
    app_id = os.environ['DEPLOY_APP_ID']
    cert_str = os.environ['DEPLOY_APP_PRIVATE_KEY']
    cert_bytes = cert_str.encode()
    private_key = default_backend().load_pem_private_key(cert_bytes, None)

    time_since_epoch_in_seconds = int(time.time())

    payload = {
        # issued at time
        'iat': time_since_epoch_in_seconds,
        # JWT expiration time (10 minute maximum)
        'exp': time_since_epoch_in_seconds + (10 * 60),
        # GitHub App's identifier
        'iss': app_id
    }

    actual_jwt = jwt.encode(payload, private_key, algorithm='RS256')

    headers = {"Authorization": "Bearer {}".format(actual_jwt),
               "Accept": "application/vnd.github.machine-man-preview+json"}

    installation_id = requests.get('https://api.github.com/app/installations', headers=headers).json()[0]['id']
    token = requests.post('https://api.github.com/app/installations/{}/access_tokens'.format(installation_id),
                          headers=headers).json()['token']
    print('GITHUB_TOKEN={}'.format(token))


if __name__ == '__main__':
    print_github_token()
