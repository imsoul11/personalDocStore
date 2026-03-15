package restapi

import "github.com/imsoul11/personalDocStore/internal/pkg/persistence"

var DefaultStore *persistence.PGStore

func SetStore(s *persistence.PGStore){
	DefaultStore = s
}