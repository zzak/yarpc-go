struct Request {
	1: required string caller
	2: required string serviceName
	3: required string encoding
	4: required string procedure

	5: optional map<string,string> headers
	6: optional string shardKey
	7: optional string routingKey
	8: optional string routingDelegate
	9: optional binary body
}
