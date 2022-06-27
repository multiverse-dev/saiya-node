package storage

type StoreCache interface {
	Store
	Persist() (int, error)
}
