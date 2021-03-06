package zk

import (
	log "github.com/funkygao/log4go"
	"github.com/funkygao/zkclient"
)

var (
	_ zkclient.ZkDataListener = &leaderElector{}
)

type leaderElector struct {
	ctx *controller

	leaderID string // participant id of the leader

	onResigningAsLeader func()
	onBecomingLeader    func()
}

func newLeaderElector(ctx *controller, onBecomingLeader func(), onResigningAsLeader func()) *leaderElector {
	return &leaderElector{
		ctx:                 ctx,
		onBecomingLeader:    onBecomingLeader,
		onResigningAsLeader: onResigningAsLeader,
	}
}

func (l *leaderElector) startup() {
	// watch for leader changes
	// leader also need watch leader changes because of network partition
	l.ctx.zc.SubscribeDataChanges(l.ctx.kb.leader(), l)
	l.elect()
}

func (l *leaderElector) fetchLeaderID() string {
	b, err := l.ctx.zc.Get(l.ctx.kb.leader())
	if err != nil {
		return ""
	}

	return string(b)
}

func (l *leaderElector) elect() (win bool) {
	// we can get here during the initial startup and the HandleDataDeleted callback.
	// because of the potential race condition, it's possible that the leader has already
	// been elected when we get here.
	l.leaderID = l.fetchLeaderID()
	if l.leaderID != "" {
		log.Trace("[%s] found leader: %s, quit elect", l.ctx.participant, l.leaderID)
		return
	}

	log.Trace("[%s] elect...", l.ctx.participant)

	if err := l.ctx.zc.CreateLiveNode(l.ctx.kb.leader(), l.ctx.participant.Marshal(), 2); err == nil {
		log.Trace("[%s] elect win!", l.ctx.participant)

		win = true
		l.leaderID = l.ctx.participant.Endpoint
		l.onBecomingLeader()
	} else {
		l.leaderID = l.fetchLeaderID() // refresh leader id
		if l.leaderID == "" {
			log.Warn("[%s] a leader has been elected but just resigned, this will lead to another round of election", l.ctx.participant)
		} else {
			log.Trace("[%s] elect lose to %s :-)", l.ctx.participant, l.leaderID)
		}
	}

	return
}

func (l *leaderElector) close() {
	// needn't delete /controller znode because when
	// zkclient closes the ephemeral znode will disappear automatically
	l.leaderID = ""
	l.ctx.zc.UnsubscribeDataChanges(l.ctx.kb.leader(), l)
}

func (l *leaderElector) amLeader() bool {
	return l.leaderID == l.ctx.participant.Endpoint
}

func (l *leaderElector) HandleDataChange(dataPath string, lastData []byte) error {
	wasLeader := l.amLeader()
	l.leaderID = l.fetchLeaderID()
	amLeader := l.amLeader()
	if wasLeader && !amLeader {
		l.onResigningAsLeader()
	}

	log.Trace("[%s] new leader is %s", l.ctx.participant, l.leaderID)
	return nil
}

func (l *leaderElector) HandleDataDeleted(dataPath string) error {
	log.Trace("[%s] leader[%s] gone!", l.ctx.participant, l.leaderID)

	if l.amLeader() {
		l.onResigningAsLeader()
	}

	l.elect()
	return nil
}
