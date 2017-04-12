package mysql

import (
	"time"

	"github.com/funkygao/dbus/engine"
	"github.com/funkygao/dbus/pkg/myslave"
	log "github.com/funkygao/log4go"
)

func (this *MysqlbinlogInput) runStandalone(dsn string, r engine.InputRunner, h engine.PluginHelper) error {
	globals := engine.Globals()
	name := r.Name()
	backoff := time.Second * 5

	this.slave = myslave.New(name, dsn, globals.ZrootCheckpoint).LoadConfig(this.cf)
	if err := this.slave.AssertValidRowFormat(); err != nil {
		panic(err)
	}

	for {
	RESTART_REPLICATION:

		log.Trace("[%s] starting replication from %s...", name, dsn)

		if img, err := this.slave.BinlogRowImage(); err != nil {
			log.Error("[%s] %v", name, err)
		} else {
			log.Trace("[%s] binlog row image=%s", name, img)
		}

		ready := make(chan struct{})
		go this.slave.StartReplication(ready)
		select {
		case <-ready:
		case <-this.stopChan:
			log.Debug("[%s] yes sir!", name)
			return nil
		}

		rows := this.slave.Events()
		errors := this.slave.Errors()
		for {
			select {
			case <-this.stopChan:
				log.Debug("[%s] yes sir!", name)
				return nil

			case err := <-errors:
				// e,g.
				// ERROR 1236 (HY000): Could not find first log file name in binary log index file
				// ERROR 1236 (HY000): Could not open log file
				log.Error("[%s] backoff %s: %v, stop from %s", name, backoff, err, dsn)
				this.slave.StopReplication()

				select {
				case <-time.After(backoff):
				case <-this.stopChan:
					log.Debug("[%s] yes sir!", name)
					return nil
				}
				goto RESTART_REPLICATION

			case pack, ok := <-r.InChan():
				if !ok {
					log.Debug("[%s] yes sir!", name)
					return nil
				}

				select {
				case err := <-errors:
					log.Error("[%s] backoff %s: %v, stop from %s", name, backoff, err, dsn)
					this.slave.StopReplication()

					select {
					case <-time.After(backoff):
					case <-this.stopChan:
						log.Debug("[%s] yes sir!", name)
						return nil
					}
					goto RESTART_REPLICATION

				case row, ok := <-rows:
					if !ok {
						log.Info("[%s] event stream closed", name)
						return nil
					}

					if row.Length() < this.maxEventLength {
						pack.Payload = row
						r.Inject(pack)
					} else {
						// TODO this.slave.MarkAsProcessed(r), also consider batcher partial failure
						log.Warn("[%s] ignored len=%d %s", name, row.Length(), row.MetaInfo())
						pack.Recycle()
					}

				case <-this.stopChan:
					log.Debug("[%s] yes sir!", name)
					return nil
				}
			}
		}
	}

	return nil
}
