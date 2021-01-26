package jobqueue_test

import (
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	badgerdb "github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/module/jobqueue"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger"
	"github.com/onflow/flow-go/utils/unittest"
)

// 0# means job at index 0 is processed.
// +1 means received a job 1
// 1! means job 1 is being processed.
// 1* means job 1 is finished

func TestConsumer(t *testing.T) {
	// [] => 										[0#]
	// on startup, if there is no more job, nothing is processed
	t.Run("testOnStartup", testOnStartup)

	// [+1] => 									[0#, 1!]
	// when received job 1, it will be processed
	t.Run("testOnReceiveOneJob", testOnReceiveOneJob)

	// [+1, 1*] => 							[0#, 1#]
	// when job 1 is finished, it will be marked as processed
	t.Run("testOnJobFinished", testOnJobFinished)

	// [+1, +2, 1*, 2*] => 			[0#, 1#, 2#]
	// when job 2 and 1 are finished, they will be marked as processed
	t.Run("testOnJobsFinished", testOnJobsFinished)

	// [+1, +2, +3, +4] => 			[0#, 1!, 2!, 3!, 4]
	// when more jobs are arrived than the max number of workers, only the first 3 jobs will be processed
	t.Run("testMaxWorker", testMaxWorker)

	// [+1, +2, +3, +4, 3*] => 	[0#, 1!, 2!, 3*, 4!]
	// when job 3 is finished, which is not the next processing job 1, the processed index won't change
	t.Run("testNonNextFinished", testNonNextFinished)

	// [+1, +2, +3, +4, 3*, 2*] => 			[0#, 1!, 2*, 3*, 4!]
	// when job 3 and 2 are finished, the processed index won't change, because 1 is still not finished
	t.Run("testTwoNonNextFinished", testTwoNonNextFinished)

	// [+1, +2, +3, +4, 3*, 2*, +5] =>	[0#, 1!, 2*, 3*, 4!, 5!]
	// when job 5 is received, it will be processed, because the worker has capacity
	t.Run("testProcessingWithNonNextFinished", testProcessingWithNonNextFinished)

	// [+1, +2, +3, +4, 3*, 2*, +5, +6] =>	[0#, 1!, 2*, 3*, 4!, 5!, 6]
	// when job 6 is received, no more worker can process it, it will be buffered
	t.Run("testMaxWorkerWithFinishedNonNexts", testMaxWorkerWithFinishedNonNexts)

	// [+1, +2, +3, +4, 3*, 2*, +5, 1*] => [0#, 1#, 2#, 3#, 4!, 5!]
	// when job 1 is finally finished, it will fast forward the processed index to 3
	t.Run("testFastforward", testFastforward)

	// [+1, +2, +3, +4, 3*, 2*, +5, 1*, +6, +7, 6*], restart => [0#, 1#, 2#, 3#, 4!, 5!, 6*, 7!]
	// when job queue crashed and restarted, the queue can be resumed
	t.Run("testWorkOnNextAfterFastforward", testWorkOnNextAfterFastforward)

	// [+1, +2, +3, ... +12, +13, +14, 1*, 2*, 3*, 5*, 6*, ...12*] => [1#, 2#, 3#, 4!, 5*, 6*, ... 12*, 13, 14]
	// when there are too many pending jobs, it will stop processing more but wait for job 4 to finish
	t.Run("testTooManyPending", testTooManyPending)

	// [+1, +2, +3, +4, Stop, 2*] => [0#, 1!, 2*, 3!, 4]
	// when Stop is called, it won't work on any job any more
	t.Run("testStopRunning", testStopRunning)

	t.Run("testConcurrency", testConcurrency)
}

func testOnStartup(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		assertProcessed(t, cp, 0)
	})
}

// [+1] => 									[0#, 1!]
// when received job 1, it will be processed
func testOnReceiveOneJob(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1

		c.Check()

		w.AssertCalled(t, []int{1})
		assertProcessed(t, cp, 0)
	})
}

// [+1, 1*] => 							[0#, 1#]
// when job 1 is finished, it will be marked as processed
func testOnJobFinished(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1

		c.Check()
		c.FinishJob(jobqueue.JobIDAtIndex(1))

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1})
		assertProcessed(t, cp, 1)
	})
}

// [+1, +2, 1*, 2*] => 			[0#, 1#, 2#]
// when job 2 and 1 are finished, they will be marked as processed
func testOnJobsFinished(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(1)) // 1*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2})
		assertProcessed(t, cp, 2)
	})
}

// [+1, +2, +3, +4] => 			[0#, 1!, 2!, 3!, 4]
// when more jobs are arrived than the max number of workers, only the first 3 jobs will be processed
func testMaxWorker(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3})
		assertProcessed(t, cp, 0)
	})
}

// [+1, +2, +3, +4, 3*] => 	[0#, 1!, 2!, 3*, 4!]
// when job 3 is finished, which is not the next processing job 1, the processed index won't change
func testNonNextFinished(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3, 4})
	})
}

// [+1, +2, +3, +4, 3*, 2*] => 			[0#, 1!, 2*, 3*, 4!]
// when job 3 and 2 are finished, the processed index won't change, because 1 is still not finished
func testTwoNonNextFinished(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3, 4})
		assertProcessed(t, cp, 0)
	})
}

// [+1, +2, +3, +4, 3*, 2*, +5] =>	[0#, 1!, 2*, 3*, 4!, 5!]
// when job 5 is received, it will be processed, because the worker has capacity
func testProcessingWithNonNextFinished(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		require.NoError(t, j.PushOne()) // +5
		c.Check()

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3, 4, 5})
		assertProcessed(t, cp, 0)
	})
}

// [+1, +2, +3, +4, 3*, 2*, +5, +6] =>	[0#, 1!, 2*, 3*, 4!, 5!, 6]
// when job 6 is received, no more worker can process it, it will be buffered
func testMaxWorkerWithFinishedNonNexts(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		require.NoError(t, j.PushOne()) // +5
		c.Check()

		require.NoError(t, j.PushOne()) // +6
		c.Check()

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3, 4, 5})
		assertProcessed(t, cp, 0)
	})
}

// [+1, +2, +3, +4, 3*, 2*, +5, 1*] => [0#, 1#, 2#, 3#, 4!, 5!]
// when job 1 is finally finished, it will fast forward the processed index to 3
func testFastforward(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		require.NoError(t, j.PushOne()) // +5
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(1)) // 1*

		time.Sleep(1 * time.Millisecond)

		w.AssertCalled(t, []int{1, 2, 3, 4, 5})
		assertProcessed(t, cp, 3)
	})
}

// [+1, +2, +3, +4, 3*, 2*, +5, 1*, +6, +7, 6*], restart => [0#, 1#, 2#, 3#, 4!, 5!, 6*, 7!]
// when job queue crashed and restarted, the queue can be resumed
func testWorkOnNextAfterFastforward(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		require.NoError(t, j.PushOne()) // +1
		c.Check()

		require.NoError(t, j.PushOne()) // +2
		c.Check()

		require.NoError(t, j.PushOne()) // +3
		c.Check()

		require.NoError(t, j.PushOne()) // +4
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(3)) // 3*
		c.FinishJob(jobqueue.JobIDAtIndex(2)) // 2*

		require.NoError(t, j.PushOne()) // +5
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(1)) // 1*

		require.NoError(t, j.PushOne()) // +6
		c.Check()

		require.NoError(t, j.PushOne()) // +7
		c.Check()

		c.FinishJob(jobqueue.JobIDAtIndex(6)) // 6*

		time.Sleep(1 * time.Millisecond)

		// rebuild a consumer with the dependencies to simulate a restart
		// jobs need to be reused, since it stores all the jobs
		reWorker := newMockWorker()
		reProgress := badger.NewChunkConsumer(db)
		reConsumer := newTestConsumer(reProgress, j, reWorker)

		err := reConsumer.Start()
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		reWorker.AssertCalled(t, []int{4, 5, 6})
		assertProcessed(t, reProgress, 3)
	})
}

// [+1, +2, +3, ... +12 , 1*, 2*, 3*, 5*, 6*, ...12*, +13, +14] => [1#, 2#, 3#, 4!, 5*, 6*, ... 12*, 13, 14]
// when there are too many pending (8) jobs, it will stop processing more but wait for job 4 to finish
// 5* - 12* has 8 pending in total
func testTooManyPending(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		for i := 1; i <= 12; i++ {
			// +1, +2, ... +12
			require.NoError(t, j.PushOne())
			c.Check()
		}

		for i := 1; i <= 12; i++ {
			// job 1 - 12 are all finished, except 4
			// 5, 6, 7, 8, 9, 10, 11, 12 are pending, which have reached max pending (8)
			if i == 4 {
				continue
			}
			c.FinishJob(jobqueue.JobIDAtIndex(i))
		}

		require.NoError(t, j.PushOne()) // +13
		c.Check()

		require.NoError(t, j.PushOne()) // + 14
		c.Check()

		time.Sleep(1 * time.Millisecond)

		// max pending reached, will not work on job 13
		w.AssertCalled(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
		assertProcessed(t, cp, 3)
	})
}

// [+1, +2, +3, +4, Stop, 2*] => [0#, 1!, 2*, 3!, 4]
// when Stop is called, it won't work on any job any more
func testStopRunning(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		for i := 0; i < 4; i++ {
			require.NoError(t, j.PushOne())
			c.Check()
		}

		c.Stop()

		c.FinishJob(jobqueue.JobIDAtIndex(2))

		time.Sleep(1 * time.Millisecond)

		// it won't work on 4 because it stopped before 2 is finished
		w.AssertCalled(t, []int{1, 2, 3})
		assertProcessed(t, cp, 0)
	})
}

func testConcurrency(t *testing.T) {
	runWith(t, func(c *jobqueue.Consumer, cp storage.ConsumerProgress, w *mockWorker, j *jobqueue.MockJobs, db *badgerdb.DB) {
		require.NoError(t, c.Start())
		// Finish job concurrently
		w.fn = func(j Job) {
			go func() {
				c.FinishJob(j.ID())
			}()
		}

		// Pushing job and checking job concurrently
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				require.NoError(t, j.PushOne())
				c.Check()
				wg.Done()
			}()
		}
		wg.Wait()

		called := make([]int, 0, 0)
		for i := 1; i <= 100; i++ {
			called = append(called, i)
		}

		time.Sleep(100 * time.Millisecond)

		w.AssertCalled(t, called)
		assertProcessed(t, cp, 100)
	})
}

type JobStatus = jobqueue.JobStatus
type Worker = jobqueue.Worker
type JobID = storage.JobID
type Job = storage.Job

func runWith(t *testing.T, runTestWith func(*jobqueue.Consumer, storage.ConsumerProgress, *mockWorker, *jobqueue.MockJobs, *badgerdb.DB)) {
	unittest.RunWithBadgerDB(t, func(db *badgerdb.DB) {
		jobs := jobqueue.NewMockJobs()
		worker := newMockWorker()
		progress := badger.NewChunkConsumer(db)
		consumer := newTestConsumer(progress, jobs, worker)
		runTestWith(consumer, progress, worker, jobs, db)
	})
}

func assertProcessed(t *testing.T, cp storage.ConsumerProgress, expectProcessed int) {
	processed, err := cp.ProcessedIndex()
	require.NoError(t, err)
	require.Equal(t, expectProcessed, processed)
}

func newTestConsumer(cp storage.ConsumerProgress, jobs storage.Jobs, worker Worker) *jobqueue.Consumer {
	log := unittest.Logger().With().Str("module", "consumer").Logger()
	maxProcessing := 3
	maxPending := 8
	c := jobqueue.NewConsumer(log, jobs, cp, worker, maxProcessing, maxPending)
	return c
}

// a Mock worker that stores all the jobs that it was asked to work on
type mockWorker struct {
	sync.Mutex
	log    zerolog.Logger
	called []Job
	fn     func(job Job)
}

func newMockWorker() *mockWorker {
	return &mockWorker{
		log:    unittest.Logger().With().Str("module", "worker").Logger(),
		called: make([]Job, 0),
		fn:     func(Job) {},
	}
}

func (w *mockWorker) Run(job Job) {
	w.Lock()
	defer w.Unlock()

	w.log.Debug().Str("job_id", string(job.ID())).Msg("worker called with job")

	w.called = append(w.called, job)
	w.fn(job)
}

// return the IDs of the jobs
func (w *mockWorker) AssertCalled(t *testing.T, expectCalled []int) {
	called := make([]int, 0)
	for _, c := range w.called {
		jobID, err := strconv.Atoi(string(c.ID()))
		require.NoError(t, err)
		called = append(called, jobID)
	}
	sort.Ints(called)
	require.Equal(t, expectCalled, called)
}
