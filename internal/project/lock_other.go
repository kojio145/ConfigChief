//go:build !windows

package project

// Off Windows the case opens without a mark (spec 14.8.2).

type lockHandle = int

const noLock = -1

func acquireLock(path string) (lockHandle, string, error) { return noLock, "", nil }

func releaseLock(h lockHandle) {}
