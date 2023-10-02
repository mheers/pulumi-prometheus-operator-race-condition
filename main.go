package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		prometheus, err := NewPrometheus(ctx)
		if err != nil {
			return err
		}

		if err := prometheus.Install(); err != nil {
			return err
		}

		return nil
	})
}
