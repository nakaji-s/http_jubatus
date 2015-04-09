# -*- coding: utf-8-unix -*-
import subprocess
import re
import json
import os

port = 3000
name = "hoge"
DEVNULL = open(os.devnull, 'wb')

client = subprocess.Popen([
    "curl",
    "localhost:{p}/classifier".format(p=port),
    "-H", 'Content-type: application/json',
    "-X", "POST",
    "-d", json.dumps({
        "name": name,
        "parameter": {
            "converter" : {
                "string_rules" : [
                    { "key" : "*", "type" : "str", "sample_weight" : "bin", "global_weight" : "bin" }
                ],
                "num_rules" : [
                    { "key" : "*", "type" : "num" }
                ]
            },
            "method" : "PA"
        }
    }),
], stdout=subprocess.PIPE, stderr=DEVNULL)  # expects foo
client.wait()

print("train1")

client = subprocess.Popen([
    "curl",
    "localhost:{p}/classifier/{n}/train".format(n=name, p=port),
    "-H", 'Content-type: application/json',
    "-X", "POST",
    "-d", json.dumps([
        [  # array
            [  # labeled datum
                "foo",  # label
                [  # datum
                    [],  # string values
                    [["hoge", 2.0]],  # num values
                    []  # binary values
                ]
            ]
        ]
    ]),
], stdout=subprocess.PIPE, stderr=DEVNULL)  # expects foo
client.wait()
print(client.stdout.read())

client = subprocess.Popen([
    "curl",
    "localhost:{p}/classifier/{n}/train".format(n=name, p=port),
    "-H", 'Content-type: application/json',
    "-X", "POST",
    "-d", json.dumps([  # array
        [
            [
                "bar",  # label
                [  # datum
                    [],  # string values
                    [["fuga", 1.0]],  # num values
                    []  # binary values
                ]
            ]
        ]
    ]),
], stdout=subprocess.PIPE, stderr=DEVNULL)  # expects foo
client.wait()
print(client.stdout.read())

print("train2")


client = subprocess.Popen([
    "curl",
    "localhost:{p}/classifier/{n}/classify".format(n=name, p=port),
    "-H", 'Content-type: application/json',
    "-X", "POST",
    "-d", json.dumps([  # array
        [
            [  # datum
                [],  # string values
                [["hoge", 1.0]],  # num value
                []  # binary values
            ]
        ]
    ]),
], stdout=subprocess.PIPE, stderr=DEVNULL)  # expects foo
client.wait()
print(client.stdout.read())
