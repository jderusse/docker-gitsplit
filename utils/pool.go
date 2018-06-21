package utils

import (
	"gopkg.in/go-playground/pool.v3"
)

type Pool struct {
	pool  pool.Pool
	batch pool.Batch
}

type PoolResult struct {
	Value func() interface{}
	Error func() error
}

type PoolResults []PoolResult

func NewPool(maxSize uint) *Pool {
	p := &Pool{
		pool: pool.NewLimited(maxSize),
	}

	p.start()

	return p
}

func (p *Pool) start() {
	p.batch = p.pool.Batch()
}

func (p *PoolResults) FirstError() error {
	for _, result := range *p {
		if err := result.Error(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pool) Wait() PoolResults {
	p.batch.QueueComplete()

	results := []PoolResult{}

	for result := range p.batch.Results() {
		results = append(results, PoolResult{
			Value: result.Value,
			Error: result.Error,
		})
	}

	p.start()

	return results
}

func (p *Pool) Close() {
	p.pool.Close()
}

func (p *Pool) Push(callback func() (interface{}, error)) {
	p.batch.Queue(func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}

		return callback()
	})
}
