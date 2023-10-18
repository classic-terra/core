#!/bin/sh
# this script is used with upgrade-test

# if $ROOT is empty, exit
if [ -z "$ROOT" ]; then
  echo "ROOT is empty, exiting..."
  exit 1
fi

# prepare .env to run astroport-core deployment testing scripts
cd $ROOT/../astroport-core/scripts

echo $(pwd)

ENV=./.env
rm $ENV
echo 'WALLET="empower father transfer coin fix auto song clean soon pet wide hamster end weapon six glass meat spice prize video repeat drift rack betray"' > $ENV
echo 'LCD_CLIENT_URL=tcp://localhost:26657' >> $ENV
echo 'CHAIN_ID=localterra' >> $ENV
echo 'TOKEN_INITIAL_AMOUNT="1000000000000000"' >> $ENV

# run astroport-core deployment testing scripts
# this 
npm run build-app

# activate ENV from running build-app


# return to root directory
cd $ROOT