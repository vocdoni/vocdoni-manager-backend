psql -U postgres  -h 172.18.0.2 -p 5432 -W -d postgres -a -f 0_init.psql 
psql -U vocdonimgr  -h 172.18.0.2 -p 5432 -W -d vocdonimanager -a -f 1_create.psql 
