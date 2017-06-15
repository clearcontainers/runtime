ui_key=ui_key.pem
ui_cert=ui_cert.pem

webui_conf_file="$ciao_bin"/webui_config.json

if [ ! -f ${ciao_pki_path}/${ui_cert} ] || [ ! -f ${ciao_pki_path}/${ui_key} ]; then
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
	    -keyout ${ciao_bin}/${ui_key} -out ${ciao_bin}/${ui_cert} \
	    -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=localhost"
    sudo install -m 0644 -t "$ciao_pki_path" ${ciao_bin}/${ui_cert} ${ciao_bin}/${ui_key}
fi

echo "Generating webui configuration file $webui_conf_file"
(
cat <<-EOF
{
    "production": {
        "controller": {
            "host": "${ciao_host}",
            "port": "${compute_api_port}",
            "protocol": "https"
        },
        "storage":{
            "host": "${ciao_host}",
            "port": "${storage_api_port}",
            "protocol": "https"
        },
        "keystone": {
            "host": "${ciao_host}",
            "port": "${keystone_admin_port}",
            "protocol": "https",
            "uri": "/v3/auth/tokens"
        },
        "ui": {
            "protocol": "https",
            "certificates": {
                "key": "${ciao_pki_path}/${ui_key}",
                "cert": "${ciao_pki_path}/${ui_cert}",
                "passphrase": "",
                "trusted": []
            }
        }
    },
    "development": {
        "controller": {
            "host": "${ciao_host}",
            "port": "${compute_api_port}",
            "protocol": "https"
        },
        "storage":{
            "host": "${ciao_host}",
            "port": "${storage_api_port}",
            "protocol": "https"
        },
        "keystone": {
            "host": "${ciao_host}",
            "port": "${keystone_admin_port}",
            "protocol": "https",
            "uri": "/v3/auth/tokens"
        },
        "ui": {
            "protocol": "https",
            "certificates": {
                "key": "${ciao_pki_path}/${ui_key}",
                "cert": "${ciao_pki_path}/${ui_cert}",
                "passphrase": "",
                "trusted": []
            }
        }
    }
}
EOF
) > $webui_conf_file