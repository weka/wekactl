FROM python:3.9-slim-buster
RUN pip3 install PyJWT==2.0.1 requests==2.25.1 cryptography==3.4.6
COPY scripts/print_github_token.py ./
RUN python --version
CMD python print_github_token.py
