# Creation

The `domain.crt` and `domain.key` files have been created using these commands:

openssl req \
  -newkey rsa:4096 \
  -nodes \
  -sha256 \
  -keyout domain.key \
  -x509 \
  -days 365 \
  -out domain.crt

You should replace these files with your own.
