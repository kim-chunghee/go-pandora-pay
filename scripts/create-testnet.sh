if [ $# -eq 0 ]; then
  echo "argument missing"
  echo "mode=normal for starting in normal mode"
  echo "mode=race to enable the race detection"
  echo "pprof to enable debugging using profiling"
  echo "debug to enable debug info"
  echo "light-computations to make the testnet use less CPU"
  exit 1
fi

SCRIPTPATH="$(
  cd -- "$(dirname "$0")" >/dev/null 2>&1
  pwd -P
)"

nodes=4
race=false
extraArgs=""

for arg in $@; do
  if [ $arg = "pprof" ]; then
    extraArgs+=" --pprof "
  fi
  if [ $arg = "mode=race" ]; then
    race=true
  fi
  if [ $arg = "debug" ]; then
    extraArgs+=" --debug "
  fi
  if [ $arg = "light-computations" ]; then
    extraArgs+=" --light-computations=true"
  fi
done

str="genesis.data,"

echo "test2 $race"

go build main.go

# Let's delete old blockchain and verify if all nodes still have a genesis file
genesisExists=true
for ((i = 0; i < $nodes; ++i)); do
  echo "deleting $i"
  rm -r ./_build/devnet_$i/DEV/logs
  rm ./_build/devnet_$i/DEV/store/blockchain_store.bolt
  rm ./_build/devnet_$i/DEV/store/mempool_store.bolt

  if [ ! -e /_build/devnet_$i/DEV/genesis.data ]; then
    genesisExists=false
  fi
done

sleep 0.2

# In case the genesis file is not found, let's create new wallets and generate the delegated stakes files
if [ genesisExists == false ]; then

  for ((i = 0; i < $nodes; ++i)); do

    echo "delete wallet $i"
    rm ./_build/devnet_$i/DEV/store/wallet_store.bolt

    echo "running $i"
    xterm -e go run main.go --instance="devnet" --instance-id="$i" --network="devnet" --wallet-derive-delegated-stake="0,0,delegated.stake" --exit
    mv ./_build/devnet_$i/DEV/delegated.stake ./_build/devnet_0/DEV/$i.stake
    echo "executed"

  done

fi

for ((i = 0; i < $nodes; ++i)); do
  str+="$i.stake"

  if [ $i != $((nodes - 1)) ]; then
    str+=","
  fi
done

# A new genesis file will be created to restart the timestamp
echo "creating genesis $str"
xterm -e go run main.go --instance="devnet" --instance-id="0" --network="devnet" --create-new-genesis="$str" --exit

sleep 0.1

echo "let's copy the genesis file to each node"
for ((i = 1; i < $nodes; ++i)); do
  echo "copying genesis $i"
  cp ./_build/devnet_0/DEV/genesis.data ./_build/devnet_$i/DEV/genesis.data
done

echo "let's delete again the blockchain to restart"
for ((i = 0; i < $nodes; ++i)); do
  rm ./_build/devnet_$i/DEV/store/blockchain_store.bolt
done

sleep 0.1

for ((i = 0; i < $nodes; ++i)); do
  echo "opening $i"
  if $race; then
    qterminal GORACE="log_path=/$SCRIPTPATH/report" -e go run -race main.go --instance="devnet" --instance-id="$i" --tcp-server-port="5230" --new-devnet --network="devnet" --set-genesis="file" --staking --hcaptcha-site-key="10000000-ffff-ffff-ffff-000000000001" --hcaptcha-secret="0x0000000000000000000000000000000000000000" --faucet-testnet-enabled="true" --delegates-allowed-enabled="true" $extraArgs &
  else
    echo 'xterm -e go run main.go --instance="devnet" --instance-id="$i" --tcp-server-port="5230" --new-devnet --network="devnet" --set-genesis="file" --staking --hcaptcha-site-key="10000000-ffff-ffff-ffff-000000000001" --hcaptcha-secret="0x0000000000000000000000000000000000000000" --faucet-testnet-enabled="true" --delegates-allowed-enabled="true" $extraArgs &'
    xterm -e go run main.go --instance="devnet" --instance-id="$i" --tcp-server-port="5230" --new-devnet --network="devnet" --set-genesis="file" --staking --hcaptcha-site-key="10000000-ffff-ffff-ffff-000000000001" --hcaptcha-secret="0x0000000000000000000000000000000000000000" --faucet-testnet-enabled="true" --delegates-allowed-enabled="true" $extraArgs &
  fi
done

wait

echo "finished"
exit 1
