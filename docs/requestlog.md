---
title: Request logging
layout: default
parent: Examples
---

# Request logging
v3.2.0
{: .label .label-yellow }
*dnspyre* can also log all DNS requests it produces. Request logging can be enabled by running *dnspyre* with flag `--log-requests`

```
dnspyre --server 8.8.8.8 google.com -n 5 --log-requests
```

The request logs will be by default available in file `requests.log`. The log file path can be configured using `--log-requests-path`.
If the file does not exist, it is created. If it exists, the logs are appended.

```
dnspyre --server 8.8.8.8 google.com -n 5 --log-requests --log-requests-path /tmp/requests.log
```

The request logs look like this:
```
2024/04/28 21:45:14 worker:[0] reqid:[37449] qname:[google.com.] qtype:[A] respid:[37449] rcode:[NOERROR] respflags:[qr rd ra] err:[<nil>] duration:[75.875086ms]
2024/04/28 21:45:14 worker:[0] reqid:[34625] qname:[google.com.] qtype:[A] respid:[34625] rcode:[NOERROR] respflags:[qr rd ra] err:[<nil>] duration:[15.643628ms]
2024/04/28 21:45:14 worker:[0] reqid:[798] qname:[google.com.] qtype:[A] respid:[798] rcode:[NOERROR] respflags:[qr rd ra] err:[<nil>] duration:[12.087964ms]
2024/04/28 21:45:14 worker:[0] reqid:[54943] qname:[google.com.] qtype:[A] respid:[54943] rcode:[NOERROR] respflags:[qr rd ra] err:[<nil>] duration:[12.975761ms]
2024/04/28 21:45:14 worker:[0] reqid:[509] qname:[google.com.] qtype:[A] respid:[509] rcode:[NOERROR] respflags:[qr rd ra] err:[<nil>] duration:[11.784968ms]
```
You can see:
* which worker executed the request (`worker`)
* what was the request and response ID (`reqid` and `respid`)
* what domain was queried and what type (`qname` and `qtype`)
* what were the DNS response flags (`respflags`)
* roundtrip duration (`duration`)
