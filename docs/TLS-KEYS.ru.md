# TLS-ключи через OpenSSL

[English](TLS-KEYS.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

Этот документ — практическое дополнение к [TLS.ru.md](TLS.ru.md): здесь
частный CA, серверный и клиентские сертификаты создаются командой
`openssl`. Канонический источник команд — официальный проект OpenSSL:

- репозиторий: <https://github.com/openssl/openssl>;
- документация: <https://docs.openssl.org/> (man-страницы `openssl-req`,
  `openssl-ca`, `openssl-x509`, `openssl-verify`).

Примеры создают цепочку на ECDSA P-256, действительную для `localhost` и
`127.0.0.1`; добавьте в SAN свои реальные имена и адреса.

## 1. Частный CA

```sh
mkdir -p /etc/cantcp/tls && cd /etc/cantcp/tls

# Ключ CA и самоподписанный сертификат (10 лет).
openssl ecparam -name prime256v1 -genkey -noout -out ca-key.pem
chmod 600 ca-key.pem
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem \
  -subj "/CN=cantcp local CA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"
```

## 2. Серверный сертификат

```sh
# Ключ сервера и CSR.
openssl ecparam -name prime256v1 -genkey -noout -out server-key.pem
chmod 600 server-key.pem
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=can-gateway.example"

# Подпись у CA; список SAN — то, что проверяет клиент.
cat > server-ext.cnf <<'EOF'
subjectAltName = DNS:can-gateway.example, DNS:localhost, IP:127.0.0.1
extendedKeyUsage = serverAuth
keyUsage = digitalSignature
EOF
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 825 -out server.pem -extfile server-ext.cnf
rm -f server.csr
```

Запуск демона: `--tls-cert server.pem --tls-key server-key.pem`. Клиент,
подключающийся по IP-адресу, должен иметь этот адрес в SAN; правьте список
SAN и перевыпускайте сертификат вместо `--tls-insecure`.

## 3. Клиентский сертификат (mutual TLS)

```sh
openssl ecparam -name prime256v1 -genkey -noout -out client-key.pem
chmod 600 client-key.pem
openssl req -new -key client-key.pem -out client.csr \
  -subj "/CN=cantcp-client-1"
cat > client-ext.cnf <<'EOF'
extendedKeyUsage = clientAuth
keyUsage = digitalSignature
EOF
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 825 -out client.pem -extfile client-ext.cnf
rm -f client.csr
```

Демон запускается с `--tls-ca ca.pem --tls-client-auth require_and_verify`,
клиент — с `--tls --tls-ca ca.pem --tls-cert client.pem --tls-key client-key.pem`.

## 4. Проверка

```sh
# Сертификат сходится к CA, ключ соответствует сертификату.
openssl verify -CAfile ca.pem server.pem client.pem
openssl pkey -in server-key.pem -pubout | diff - <(openssl x509 -in server.pem -pubkey -noout)

# Что проверит настоящий клиент (включая имя хоста).
openssl s_client -connect can-gateway.example:29536 -tls1_3 \
  -CAfile ca.pem -servername can-gateway.example </dev/null

# Версия рукопожатия и субъект сертификата пира.
openssl s_client -connect can-gateway.example:29536 -tls1_3 -CAfile ca.pem </dev/null 2>/dev/null \
  | grep -E "Protocol|subject="
```

## 5. Использование файлов

- демон: `--tls-cert server.pem --tls-key server-key.pem` (плюс `--tls-ca` и
  `--tls-client-auth` для mTLS); пути можно хранить в JSON-конфигурации
  вместо командной строки;
- CLI: `--tls --tls-ca ca.pem [--tls-cert client.pem --tls-key client-key.pem]`;
- **программы на Go** с `cantcp-lib-go`: пара загружается
  `tls.LoadX509KeyPair`, CA — `x509.NewCertPool`; полный пример сервера и
  клиента — в репозитории библиотеки (`examples/tls/`);
- **программы на Python** с `cantcp-lib-python`: пути передаются в
  `ssl.SSLContext.load_cert_chain` и `load_verify_locations`; полный пример —
  в репозитории библиотеки (`examples/tls/`).

## 6. Ротация

Замените серверные файлы и перечитайте демон (`systemctl reload cantcpd`,
см. [TLS.ru.md](TLS.ru.md)). CA и клиентские сертификаты читаются при
запуске клиента: раздайте новые материалы и перезапустите клиентов. Ключ CA
держите offline: он нужен только при выпуске сертификатов.
