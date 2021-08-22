package manager

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestCreateHostedSquad(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	if err = m.CreateSquad("", "0xff", "lolo", "test squad", PRIVATE, "lolo2001", HOSTED, "lolo"); err != nil {
		t.Error(err)
	}
}

func TestCreateSquad(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	if err = m.CreateSquad("", "0xfg", "lolo", "test squad", PRIVATE, "lolo2001", MESH, "lolo"); err != nil {
		t.Error(err)
	}
}

func TestCreatePeer(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	_ = m
	// if err = m.P("0xff", "lolo", "test squad", PRIVATE, "lolo2001", HOSTED, "lolo"); err != nil {
	// 	t.Error(err)
	// }
}

func TestConnectSquad(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	if err = m.ConnectToSquad("", "0xff", "lolo3", "lolo2001", HOSTED); err != nil {
		t.Error(err)
		return
	}
}

func TestLeaveSquad(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	if err = m.LeaveSquad("0xff", "lolo", HOSTED); err != nil {
		t.Error(err)
		return
	}
}

func TestListAllSquads(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	squads, err := m.ListAllSquads(0, MESH)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
	squads, err = m.ListAllSquads(0, HOSTED)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
}

func TestListSquadsByID(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	squads, err := m.ListSquadsByID(0, "g", MESH)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
	squads, err = m.ListSquadsByID(0, "xf", HOSTED)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
}

func TestListSquadsByName(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	squads, err := m.ListSquadsByName(0, " squad", MESH)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
	squads, err = m.ListSquadsByName(0, "squad", HOSTED)
	if err != nil {
		t.Error(err)
	}
	t.Error(squads)
}

func TestPeerCreate(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	ec, err := m.PeerAuthInit("lolo_test_2")
	if err != nil {
		t.Error(err)
		return
	}
	privKey, err := m.AuthManager.parsePrivKey(PRIV_KEY)
	if err != nil {
		t.Error(err)
		return
	}
	res, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, []byte(ec))
	if err != nil {
		t.Error(err)
		return
	}
	if err = m.PeerAuthVerif("lolo_test_2", res); err != nil {
		t.Error(err)
		return
	}
}

func TestPeerCall(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Error(err)
		return
	}
	err = m.AddIncomingCall("lolo_test", "Patrick")
	if err != nil {
		t.Error(err)
		return
	}
}
