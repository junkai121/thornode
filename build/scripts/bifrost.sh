#!/bin/sh
set -ex

CHAIN_ID="${CHAIN_ID:=thorchain}"
BINANCE_HOST="${BINANCE_HOST:=https://data-seed-pre-0-s3.binance.org}"
DB_PATH="${DB_PATH:=/var/data}"
CHAIN_API="${CHAIN_API:=127.0.0.1:1317}"
CHAIN_RPC="${CHAIN_RPC:=127.0.0.1:26657}"
SIGNER_NAME="${SIGNER_NAME:=thorchain}"
SIGNER_PASSWD="${SIGNER_PASSWD:=password}"
START_BLOCK_HEIGHT="${START_BLOCK_HEIGHT:=1}"
TSS_SCHEME="${TSS_SCHEME:=http}"
TSS_HOST="${TSS_HOST:=127.0.0.1}"
TSS_PORT="${TSS_PORT:=4040}"

$(dirname "$0")/wait-for-thorchain-api.sh $CHAIN_API

echo "PEER: $PEER"
if [ ! -z "$PEER" ]; then
    echo "got here"
    PEER="/ip4/$PEER/tcp/5040/ipfs/$(curl http://$PEER:6040/p2pid)"
fi

OBSERVER_PATH=$DB_PATH/bifrost/observer/
SIGNER_PATH=$DB_PATH/bifrost/signer/

mkdir -p $SIGNER_PATH $OBSERVER_PATH /etc/bifrost

# Generate bifrost config file
echo "{
    \"thorchain\": {
        \"chain_id\": \"$CHAIN_ID\",
        \"chain_host\": \"$CHAIN_API\",
        \"signer_name\": \"$SIGNER_NAME\"
    },
    \"metrics\": {
        \"enabled\": true
    },
    \"chains\": [
      {
        \"chain_id\": \"BNB\",
        \"rpc_host\": \"$BINANCE_HOST\",
        \"block_scanner\": {
          \"rpc_host\": \"$BINANCE_HOST\",
          \"enforce_block_height\": false,
          \"block_scan_processors\": 1,
          \"block_height_discover_back_off\": \"1s\",
          \"block_retry_interval\": \"10s\",
          \"chain_id\": \"BNB\",
          \"http_request_timeout\": \"30s\",
          \"http_request_read_timeout\": \"30s\",
          \"http_request_write_timeout\": \"30s\",
          \"max_http_request_retry\": 10,
          \"start_block_height\": 0,
          \"db_path\": \"$OBSERVER_PATH\"
        }
      }
    ],
    \"tss\": {
        \"scheme\": \"$TSS_SCHEME\",
        \"host\": \"$TSS_HOST\",
        \"port\": $TSS_PORT,
        \"bootstrap_peers\": [
          \"$PEER\"
        ],
        \"rendezvous\": \"asgard\",
        \"p2p_port\": 5040,
        \"info_address\": \":6040\",
        \"tss_address\": \":4040\"
    },
    \"signer\": {
      \"signer_db_path\": \"$SIGNER_PATH\",
      \"block_scanner\": {
        \"rpc_host\": \"$CHAIN_RPC\",
        \"start_block_height\": $START_BLOCK_HEIGHT,
        \"enforce_block_height\": false,
        \"block_scan_processors\": 1,
        \"block_height_discover_back_off\": \"1s\",
        \"block_retry_interval\": \"10s\",
        \"scheme\": \"http\"
      }
    }
}" > /etc/bifrost/config.json
echo "7b225061696c6c696572534b223a7b224e223a32353639323032383233323836323833313437323030303234333236353934393734383039303935313136393138363939383935323735343539323736363837383633373535393132393536393737323739393438333030303132383338343437383635363435343434383635303235353731323036353131373737323137353530343134373038323936393533323136323634323035303434343938313935303533383639353132353031323033393536393535343437333931333038383438323830393038313533363736323338323133313539343530333430383531303231303630303636353634333031363932313734393738383139383736323537363638323535383832333734393538333330363530383135333437353634343932313733373434313337333036393731303837313632373736303836363138363733333430353136323135323331363132363537333735343830353434353535303733303539353135343034353639373730373434393332373431353337343038373332303435343933393936393538353530373430373737303738383134333635333339383036363830363132323630313138323034373430363631313336353236373438393635333331393535333537363632343635393835313335383232333936323939323234323436303139393039373636343935363336303435313838313134303232383034343930353230323132393232373034393233343133363830333033373934303832313131303736323537373531303035323230363731393938303631373335343237373934383034373339393537373336373832373330353837343037332c224c616d6264614e223a31323834363031343131363433313431353733363030303132313633323937343837343034353437353538343539333439393437363337373239363338333433393331383737393536343738343838363339393734313530303036343139323233393332383232373232343332353132373835363033323535383838363038373735323037333534313438343736363038313332313032353232323439303937353236393334373536323530363031393738343737373233363935363534343234313430343534303736383338313139313036353739373235313730343235353130353330303333323832313530383436303837343839343039393338313238383334313237393431313837343739313635333235343037363733373832323436303836383732303638363533343835353433353635323632333539333233363731313930353138333638383831393739353938373636323833363635303433373235343239343435323833353430363937343138383135383638343039383639393032363934333537333938383730363435343839313936373438343237303239373930313633353031323030353632303539343332373438343233393933303937343131323434303330393232343032343134393532333135343735303237353931393439333633333735383237323337363030383136383534393031383339313533313036383038343035393636313532343636323834313935393035333134353734333533313934303730393734303039303436383836343438313839383734323432333730363438343333363039373439373237333430393538363932333637373233342c225068694e223a32353639323032383233323836323833313437323030303234333236353934393734383039303935313136393138363939383935323735343539323736363837383633373535393132393536393737323739393438333030303132383338343437383635363435343434383635303235353731323036353131373737323137353530343134373038323936393533323136323634323035303434343938313935303533383639353132353031323033393536393535343437333931333038383438323830393038313533363736323338323133313539343530333430383531303231303630303636353634333031363932313734393738383139383736323537363638323535383832333734393538333330363530383135333437353634343932313733373434313337333036393731303837313330353234373138363437333432333831303336373337373633393539313937353332353637333330303837343530383538383930353637303831333934383337363331373336383139373339383035333838373134373937373431323930393738333933343936383534303539353830333237303032343031313234313138383635343936383437393836313934383232343838303631383434383034383239393034363330393530303535313833383938373236373531363534343735323031363333373039383033363738333036323133363136383131393332333034393332353638333931383130363239313438373036333838313431393438303138303933373732383936333739373438343834373431323936383637323139343939343534363831393137333834373335343436387d2c224e54696c646569223a32333730393131363633323138383635353734393837373137373736363937313734333534343734393038303834343539333639353539383938303234303335393937303036353235333536363538303631353737373631393231363635323233393434373936303639333839393130393330393738383632363134393738333035373433333936343830303633383135363933333338313935343634383435313936353236353432353634333839393537313130363731393139343832303735383131393832323831303038313437363331353830383736323330323030363136313038323330313130353337353132333333353639343333343634303231333932353232323137353331393737323638333835303234363933303938313732333730383637353839313630363136353434323530383733373639383038323331353231313438333732333936343738343538383633333430363932363131343031393438373735333035363034363233333534363933363135373638393332333735393432313538353130333837313733383232383933343636393435373238313039333632393137323638393330343735363039333534353736383739343536333336393030373636303438393336393537333733353535393436343038383639373931333939353733303633383631393837353932383437363731303536393231323836303234313030303036353435323434373936393035313731303931313935363039383338323538333039353933363631333834313737333437393834383630343834353832343833343830353531373737373338353139303339383631343834392c22483169223a32313638363734383232373934353936343434353231393730343039393431313239323339363630363037383231353939353937363639343638333436343436323036353930373737383732383436313138303835383131333038393632313332313735393030383132383131393232333432373739363132303532353639393930393330393330393436323332393536373239313033353233303636343530353633383339333735363137353932303934333635333039363430333134373130373037393831393133333033383137313638343237363339353139303439343438353739363137383333363536343335333131353037383337333430323538363531373838383736343638323738333736343134373731373734303935323230303438373435383238323434393137373838323238393932383335343133303033373937303335363538363532323932333336323431343433363330373434383631333737303639363233323130373534313535353433313030303133393236373639323634363038343730383234393235323636353137343931393932313434303533333338303635373438343134363633303631313134363839333231343531303836383338343230313530313231353436333438343135343230383032393030333138303035373930343139343235313632373132333931313138383732323739313135373736383538333531353232373338373239383337393539303232363232373530333035363935363239353936353533393331383232353235343137393431363835323737363533363532393333303635393534393737353139303037303939322c22483269223a31393033303934333837313930393332333932303839313530363031373136313433363136333433343233313633303033343130353834373534323630333931363133333434373334383338343432313031363934383530363932303432383231333831353530313433363538363834373038383235373733313231383530303731383039343732303333383536313331343236353138353436353930363535353332363533383033363639383234313437343530323531373433393537353638353930333938353238343133303539363836303332363336313830363235333435353535343538343638323130313830333430323633383938303538333837363635313338303939333136383431303332343930333934303433313930323439343135313733363931303938303134363937343839303134383638353238303730343533303031393834303038323736323931313931303238373633313930323637373131333535313938383931313631313032373731333831353830383738373332363434363632343335313234343838343833313735343134323236303233313232313733363631333133383531383439323534373830373032333230313338343833343431353538363935323530333030373936363835353538303139363733313134353932303834373632363136333435303335303337333830323433313535333837333932303836353839353438323530303336363937343139323636383732353538383136393631373338393636323134363633343037303038323934323239393530303631363936313436323131393539343433303233393036363739383933327d" >/etc/bifrost/preparam.data
export SIGNER_PASSWD
exec "$@"
