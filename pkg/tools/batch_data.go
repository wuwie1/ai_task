package tools

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Config struct {
	MaxThread                int
	CacheNum                 int
	TimeIntervalMilliSeconds int64
}

func GetDefaultConfig() *Config {
	return &Config{
		MaxThread:                100,
		CacheNum:                 200,
		TimeIntervalMilliSeconds: 500,
	}
}

type Processor struct {
	Name          string
	config        *Config
	messageChan   chan interface{}
	isOpen        bool
	cacheChan     chan interface{}
	cacheChanLock sync.Mutex
	threadChan    chan struct{}
	ctx           context.Context
	cancelFunc    context.CancelFunc
	messageWg     sync.WaitGroup
	serviceLock   sync.RWMutex
	updateTime    int64
	handler       func(batchData []interface{}) error
}

func NewProcessor(name string, config *Config, handler func(batchData []interface{}) error) *Processor {
	return &Processor{
		Name:        name,
		config:      config,
		handler:     handler,
		messageChan: make(chan interface{}, 1024),
	}
}

func (p *Processor) Start() {

	p.serviceLock.Lock()
	defer p.serviceLock.Unlock()

	if p.isOpen {
		return
	}

	p.threadChan = make(chan struct{}, p.config.MaxThread)
	p.cacheChan = make(chan interface{}, p.config.CacheNum)

	p.ctx, p.cancelFunc = context.WithCancel(context.Background())

	p.updateCacheDataChan()

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				p.messageWg.Wait()
				return
			case msg := <-p.messageChan:
				p.process(msg)
			}
		}

	}()

	p.isOpen = true

}

func (p *Processor) Stop() {

	p.serviceLock.Lock()
	defer p.serviceLock.Unlock()

	if !p.isOpen {
		return
	}

	p.cancelFunc()

	p.messageWg.Wait()

	p.isOpen = false

}

func (p *Processor) updateCacheDataChan() {

	go func() {

		logrus.Infof("batch process: %s batch handle thread start", p.Name)
		defer logrus.Infof("batch process: %s batch handle thread close", p.Name)

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(time.Millisecond * time.Duration(p.config.TimeIntervalMilliSeconds)):

				p.cacheChanLock.Lock()
				if time.Now().UnixNano()-p.updateTime > p.config.TimeIntervalMilliSeconds*1000000 ||
					time.Now().UnixNano()-p.updateTime < 0 {

					var dataSlice []interface{}

					for {
						select {
						case cacheData := <-p.cacheChan:
							dataSlice = append(dataSlice, cacheData)
							continue
						default:
						}
						break
					}

					if len(dataSlice) == 0 {
						p.cacheChanLock.Unlock()
						continue
					}

					p.threadChan <- struct{}{}
					p.messageWg.Add(1)
					go func(batchData []interface{}) {
						defer func() {
							p.messageWg.Done()
							<-p.threadChan
						}()

						err := p.handler(batchData)
						if err != nil {
							logrus.Errorf("batch process: %s batch handle err: %s", p.Name, err.Error())
							return
						}

					}(dataSlice)

					//update time
					p.updateTime = time.Now().UnixNano()
				}

				p.cacheChanLock.Unlock()

			}

		}

	}()

}

func (p *Processor) GetMessageChan() chan interface{} {
	return p.messageChan
}

func (p *Processor) process(data interface{}) {
	p.cacheChanLock.Lock()
	defer p.cacheChanLock.Unlock()

	select {
	case p.cacheChan <- data:
		return
	default:
		//update time
		defer func() {
			p.updateTime = time.Now().UnixNano()
		}()

		var dataSlice []interface{}

		//add last data
		dataSlice = append(dataSlice, data)

		for {
			select {
			case cacheData := <-p.cacheChan:
				dataSlice = append(dataSlice, cacheData)
				continue
			default:
			}
			break
		}

		p.threadChan <- struct{}{}
		p.messageWg.Add(1)

		go func(batchData []interface{}) {
			defer func() {
				p.messageWg.Done()
				<-p.threadChan
			}()

			err := p.handler(batchData)
			if err != nil {
				logrus.Errorf("batch process: %s batch handle err: %s", p.Name, err.Error())
				return
			}

		}(dataSlice)

	}
}
