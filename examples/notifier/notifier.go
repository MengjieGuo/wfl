package main

import (
	"fmt"

	"github.com/dgruber/wfl"
)

func main() {

	notifier := wfl.NewNotifier()

	go func() {
		// pre-proc followed by parallel exec
		wfl.NewJob(wfl.NewWorkflow(wfl.NewProcessContext())).
			TagWith("A").
			Run("sleep", "1").
			ThenRun("sleep", "3").
			Run("sleep", "2").
			Synchronize().
			Notify(notifier)
	}()

	go func() {
		// pre-proc followed by parallel exec
		wfl.NewJob(wfl.NewWorkflow(wfl.NewProcessContext())).
			TagWith("B").
			Run("sleep", "1").
			ThenRun("sleep", "2").
			Run("sleep", "2").
			Synchronize().
			Notify(notifier)
	}()

	job1 := notifier.ReceiveJob()
	fmt.Printf("finished with sequence: %s\n", job1.Tag())

	job2 := notifier.ReceiveJob()
	fmt.Printf("finished with sequence: %s\n", job2.Tag())

}
