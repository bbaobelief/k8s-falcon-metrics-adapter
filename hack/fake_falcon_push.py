#!-*- coding:utf8 -*-

import requests
import time
import json
import random


ts = int(time.time())


v = int(1048576+random.randint(1000,5000))

# 通过kubectl get pods -A 获取pod name

payload = [
    {
        "endpoint": "test-sample-59f6b8979b-dcsdn",
        "metric": "cpu.busy",
        "timestamp": ts,
        "step": 60,
        "value": random.randint(20,50),
        "counterType": "GAUGE",
        "tags": "",
    },

    {
        "endpoint": "test-sample-59f6b8979b-dcsdn",
        "metric": "net.if.in.bytes",
        "timestamp": ts,
        "step": 60,
        "value": v,
        #"counterType": "DERIVE",
        "counterType": "GAUGE",
        "tags": "iface=eth0",
    },
]

print(payload)

r = requests.post("http://127.0.0.1:1988/v1/push", data=json.dumps(payload))

print r.text