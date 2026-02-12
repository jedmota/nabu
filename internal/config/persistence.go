package config

// DefaultPersistence implements viewmodel.ConfigPersistence by delegating
// to the existing package-level functions that operate on the filesystem.
type DefaultPersistence struct{}

func (DefaultPersistence) LoadWhitelist() ([]WhitelistPattern, error)                  { return LoadWhitelist() }
func (DefaultPersistence) AddToWhitelist(pattern string) error                         { return AddToWhitelist(pattern) }
func (DefaultPersistence) RemoveFromWhitelist(pattern string) error                    { return RemoveFromWhitelist(pattern) }
func (DefaultPersistence) EditWhitelistPattern(oldPattern, newPattern string) error     { return EditWhitelistPattern(oldPattern, newPattern) }
func (DefaultPersistence) ToggleWhitelistPattern(pattern string) error                 { return ToggleWhitelistPattern(pattern) }
func (DefaultPersistence) ClearWhitelist() error                                       { return ClearWhitelist() }
func (DefaultPersistence) LoadMapLocal() ([]MapLocalEntry, error)                      { return LoadMapLocal() }
func (DefaultPersistence) AddMapLocalEntry(entry MapLocalEntry) error                  { return AddMapLocalEntry(entry) }
func (DefaultPersistence) RemoveMapLocalEntry(pattern string) error                    { return RemoveMapLocalEntry(pattern) }
func (DefaultPersistence) ToggleMapLocalEntry(pattern string) error                    { return ToggleMapLocalEntry(pattern) }
func (DefaultPersistence) LoadMapRemote() ([]MapRemoteEntry, error)                    { return LoadMapRemote() }
func (DefaultPersistence) AddMapRemoteEntry(entry MapRemoteEntry) error                { return AddMapRemoteEntry(entry) }
func (DefaultPersistence) RemoveMapRemoteEntry(pattern string) error                   { return RemoveMapRemoteEntry(pattern) }
func (DefaultPersistence) ToggleMapRemoteEntry(pattern string) error                   { return ToggleMapRemoteEntry(pattern) }
func (DefaultPersistence) UpdateMapRemoteEntry(oldPattern string, entry MapRemoteEntry) error {
	return UpdateMapRemoteEntry(oldPattern, entry)
}
