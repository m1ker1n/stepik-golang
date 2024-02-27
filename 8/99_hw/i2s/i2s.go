package main

import (
	"errors"
	"reflect"
)

var (
	errNotReferenceType = errors.New("out should be reference type")
	errCantSet          = errors.New("can't set value")
	errCantAssertData   = errors.New("can't assert data")
	errCantConvert      = errors.New("can't convert to needed type")
)

func i2s(data interface{}, out interface{}) error {
	return i2sReflectRecursive(data, reflect.ValueOf(out))
}

func i2sReflectRecursive(data any, out reflect.Value) error {
	switch out.Kind() {
	case reflect.Pointer, reflect.Slice:
	default:
		return errNotReferenceType
	}

	v := out.Elem()
	t := v.Type()
	var toSet reflect.Value
	if !v.CanSet() {
		return errCantSet
	}

	switch v.Kind() {
	case reflect.Struct:
		dataMap, ok := (data).(map[string]any)
		if !ok {
			return errCantAssertData
		}

		toSet = reflect.New(v.Type()).Elem()
		for i := range v.NumField() {
			fieldName := v.Type().Field(i).Name
			fieldValuePtr := reflect.New(v.Field(i).Type())
			err := i2sReflectRecursive(dataMap[fieldName], fieldValuePtr)
			if err != nil {
				return err
			}
			toSet.Field(i).Set(fieldValuePtr.Elem())
		}

	case reflect.Slice:
		dataSlice, ok := (data).([]any)
		if !ok {
			return errCantAssertData
		}

		toSet = reflect.MakeSlice(v.Type(), len(dataSlice), cap(dataSlice))
		for i, dataEl := range dataSlice {
			elToFill := reflect.New(toSet.Type().Elem())
			err := i2sReflectRecursive(dataEl, elToFill)
			if err != nil {
				return err
			}
			toSet.Index(i).Set(elToFill.Elem())
		}

	default:
		if !reflect.TypeOf(data).ConvertibleTo(t) {
			return errCantConvert
		}
		toSet = reflect.ValueOf(data).Convert(t)
	}

	v.Set(toSet)
	return nil
}
