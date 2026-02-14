package dispatcher

import "reflect"

type DeviceDispatcher func(
	deviceId string,
	methodName string,
	args ...interface{},
) []reflect.Value
