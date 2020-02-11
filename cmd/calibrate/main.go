package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/BTBurke/monny/pkg/metric"
	"github.com/BTBurke/monny/pkg/rng"
	"github.com/BTBurke/monny/pkg/stat"
)

const (
	NumProcs int     = 4
	Loops    int     = 10000
	Run      int     = 100000
	Cap      int     = 100
	Lambda   float64 = 0.25
)

var wg sync.WaitGroup

type results struct {
	name string
	mu   sync.Mutex
	val  map[float64]float64
}

func (r *results) record(k float64, type1error float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.val[type1error] = k
}

func newResults(name string) *results {
	return &results{
		name: name,
		val:  make(map[float64]float64),
	}
}

func main() {
	res := newResults("ln-ewma")
	start := time.Now()
	for k := 5.0; k <= 7.0; k += 0.1 {
		wg.Add(1)
		log.Printf("start k=%f\n", k)
		go errorRate(res, k)
	}
	wg.Wait()
	fmt.Printf("Time Elapsed: %v\n", time.Since(start))
	var b bytes.Buffer
	for lambda, e := range res.val {
		b.WriteString(fmt.Sprintf("%f %f\n", lambda, e))
	}
	ioutil.WriteFile(fmt.Sprintf("%s.txt", res.name), b.Bytes(), 0644)
}

func errorRate(results *results, k float64) {
	defer wg.Done()
	errors := 0
	for i := 0; i < Loops; i++ {
		s, err := stat.NewEWMATestStatistic("ewma", 0.25, k, stat.NewLogNormal(Cap))
		if err != nil {
			log.Fatalf("unexpected error contructing test statistic: %v", err)
		}

		t, err := stat.NewLogNormalTest(metric.NewName("calibrate", nil), stat.WithLogNormalStatistic(s))
		if err != nil {
			log.Fatalf("unexpected error constructing test statistic: %v", err)
		}
		rng := rng.NewLogNormalRNG(5.0, 1.0)

		for j := 0; j < Cap; j++ {
			if err := t.Record(rng.Rand()); err != nil {
				log.Fatalf("unexpected error recording value: %v", err)
			}
		}
		for j := 0; j < Run; j++ {
			_ = t.Record(rng.Rand())
			if t.HasAlarmed() {
				errors++
				break
			}
		}
	}
	type1error := float64(errors) / float64(Loops)
	fmt.Printf("Result: k=%1.1f p=%1.5f errs=%d\n", k, type1error, errors)
	results.record(k, type1error)
}
