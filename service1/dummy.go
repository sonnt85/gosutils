// +build openbsd, netbsd

package service

type Service struct {
}

func NewService(name string, autoinstall bool, description string, dependencies []string, args ...string) *Service {
	return nil
}

func (srv *Service) GetTemplate() (rettr string) {
	return
}
func (srv *Service) SetTemplate(arg string) (err error) {
	return
}

func (srv *Service) Install(args ...string) (restr string, err error) {
	return
}

func (srv *Service) Remove() (restr string, err error) {
	return
}

func (srv *Service) Start() (restr string, err error) {
	return
}

func (srv *Service) Stop() (restr string, err error) {
	return
}

func (srv *Service) Status() (restr string, err error) {
	return
}

func (srv *Service) Run(e string) (restr string, err error) {
	return
}
