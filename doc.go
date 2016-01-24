// Copyright (c) 2016 Mariano Cano. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package retry simplifies the retry of code when an error occurs.

With retry, automatic retries can be written as simple as:

	executor := retry.NewExecutor()
	err := executor.Execute(func() error {
		return methodToRetry()
	}

The code above create a default retry.Executor configured to retry the code once
after one second. But it can be configured with different number of retries,
different backoff strategies and different conditions to retry, all of them
based on the return of the function executed.

To use different backoff strategies we must add them to the retry.Executor:

	executor := retry.NewExecutor.
		WithBackoff(retry.ExponentialDelayBackoff(100 * time.Milliseconds, 2))

Among others, the package retry includes the following backoff strategies:

	retry.FixedDelayBackoff(delay time.Duration)
	retry.UniformRandomBackoff(maxDelay time.Duration)
	retry.ExponentialDelayBackoff(initialDelay time.Duration, multiplier float64)
	retry.BoundedDelayBackoff(minDelay, maxDelay time.Duration)

A user can also write a custom backoff strategy implementation using the
interface retry.Backoff

	type Backoff interface {
		GetDelay(Context) time.Duration
	}

The method GetDelay returns just the time.Duration that the code will sleep
using time.Sleep(d) before retrying again. The Context passed provides some
properties like the number of retries or the last delay.

The retry.Executor can be modified to only retry specific errors and panics with:

	executor.WithError(err error)
	executor.WithErrorComparator(f func(error) bool)
	executor.WithPanic()

And the backoff strategies can be modified with:

	executor.WithFirstRetryNoDelay()
	executor.WithFixedJitter(delay time.Duration)
	executor.WithMaxDelay(maxDelay time.Duration)
	executor.WithMinDelay(minDelay time.Duration)
	executor.WithProportionalJitter(multiplier float64)
	executor.WithRandomJitter(rangeDelay time.Duration)
	executor.WithUniformJitter(rangeDelay time.Duration)

It's also possible to use the ErrorChannel field in the retry.Executor struct
to retrieve all the errors. This is a buffered channel with the number of
retries + 1. If this number is not set and the channel is not read a deadlock
can occur. To enable the ErrorChannel use:

	executor.WithErrorChannel()

All methods in retry.Executor can be chained together, the following code will
create an executor that will retry the code twice, only with the error
ErrInternalServerError, it will wait at least 500ms and at most 1500ms:

	executor := retry.NewExecutor().
		WithRetries(2).
		WithError(ErrInternalServerError).
		WithDelay(500 * time.Milliseconds).
		WithRandomJitter(1 * time.Second).
		WithMinDelay(500 * time.Milliseconds)

*/
package retry
