package api;

import lombok.Data;

import java.util.Map;

@Data
public class Node {
    private String node_id;
    private String node_type;
    private String node_name;
    private String node_input;
    private String node_output;
    private String chain_root_id;
    private String chain_id;
    private int step;
    private boolean has_children;
    private Map<String, Object> inputs;
    private String prompt;
    private String response;
    private long creation_timestamp;
    private String finish_reason;
    private long latency;
    private int query_input_tokens;
    private int query_output_tokens;
    private int query_total_tokens;
    private Map<String, Object> params;
    private String target;
}
