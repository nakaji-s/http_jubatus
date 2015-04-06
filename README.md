# HTTP Jubatus

This is thin wrapper of jubatus rpc for http request.

## usage

boot server.

```
$ go run http_jubatus.go
```

you can create new jubatus process via rpc.

```
$ curl localhost:3000/classifier \
    -H 'Content-type: application/json' \
    -X POST \
    -d '{ \
          "name": "sample_classifier", \
          "parameter": { \
            "converter" : { \
                "string_rules" : [ \
                    { "key" : "*", type : "str", "sample_weight" : "bin", "global_weight" : "bin" }\
                ],\
                "num_rules" : [ \
                    { "key" : "*", type : "num" } \
                ]\
            },\
            "method" : "PA"\
        }\
        }'
```

you will get result

```
ok
```

For detail of jubatus configuration, you can see [documentation of jubatus](http://jubat.us/en/)
More examples of jubatus configuration, you can see [jubatus repository](https://github.com/jubatus/jubatus/tree/master/config).

Now, you can access HTTP endpoint `/classifier/sample_classifier/<method>`.

```
$ curl localhost:3000/classifier/sample_classifier/train \
    -H 'Content-type: application/json' \
    -X POST \
    -d [ \  # arguments must be passed as array
        [ \  # classify method requires array of `labeled_datum`
            [ \  # a `labeled datum` structure consist of 2-length array
                "bar" \  # label
                [ \  # datum
                    [], \  # string_values(it is array)
                    [ \  # num_values(it is array)
                        ["fuga", 1.0] \ # num_value
                    ], \
                    [] \ # binary_valies (it is array)
                ] \
            ] \
        ] \
      ]
```

you will get result

```
{"result": 1}
```

For details of methods, you can see [jubatus documentation](http://jubat.us/en/api.html).
Definition of interface, you can see interface definition in *.idl files in [this page](https://github.com/jubatus/jubatus/tree/master/jubatus/server/server)

## license

MIT License
