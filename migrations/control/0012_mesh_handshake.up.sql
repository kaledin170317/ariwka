UPDATE node_inbounds
SET server_name = 'www.cloudflare.com'
WHERE network = 2 AND tag LIKE 'mesh:%' AND server_name = 'microsoft.com';
