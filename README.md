# Clockify Weekly Report Sender

## Configure
Edit the file `cwrs.json` and fill in the `MISSING` fields. You will need an email that allows SMTP submissions.
You can use Gmail for this, follow [this guide](https://support.google.com/accounts/answer/6010255?hl=en).

## Usage
```bash
docker-compose up
```
With the two files, `docker-compose.yml` and `cwrs.json` in the same directory.

If you haven't made another docker user, you can use `root.crontab` to make the report sent out weekly.
```bash
00 04 * * 1 /bin/bash /root/cwrs/root.crontab
```