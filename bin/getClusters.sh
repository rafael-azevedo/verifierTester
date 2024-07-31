page=1
output_file="byovpcClusters.txt"
if test -f $output_file; then 
    echo "Removing old $output_file"
    rm $output_file
fi

while true; do
    response=$(ocm get /api/clusters_mgmt/v1/clusters --parameter page=$page --parameter search="cloud_provider.id='aws' AND state='ready' AND hypershift.enabled='false'")
    size=$(jq -r '.size' <<< "$response")    
    if [[ $size -eq 0 ]]; then
        echo "DONE"
        break
    fi

    # Use jq to extract the desired fields and print each line with a newline at the end
    jq -r '.items[] | "\(.id) \(.name) \(.aws.subnet_ids)"' <<< "$response" | while IFS= read -r line; do
        subnet_ids=$(jq -r --arg line "$line" '.items[] | select("\(.id) \(.name) \(.aws.subnet_ids)" == $line) | .aws.subnet_ids' <<< "$response")
        if [[ $subnet_ids != "null" ]]; then
            printf '%s\n' "$line" >> "$output_file"
        fi
    done
    page=$((page + 1))
done

