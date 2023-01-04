Generally we should avoid shell scripting and write tests purely in Golang.
However, some libraries are not Goroutine-safe (e.g. app simulations cannot be run safely in parallel),
and OS-native threading may be more efficient for many parallel simulations, so we use shell scripts here.

# For quick node testing

```
bash scripts/run-node.sh <terrad binary> <denom>
```

If "terrad binary" and "denom" is left blank, it will build current terrad and set denom to uluna.
```
bash scripts/run-node.sh
```