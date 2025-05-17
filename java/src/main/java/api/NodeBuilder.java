package api;

import java.util.Map;

public class NodeBuilder {
    private String nodeId;
    private String nodeType;
    private String nodeName;
    private String nodeInput;
    private String nodeOutput;
    private String chainRootId;
    private String chainId;
    private int step;
    private boolean hasChildren;
    private Map<String, Object> inputs;
    private String prompt;
    private String response;
    private long creationTimestamp;
    private String finishReason;
    private long latency;
    private int queryInputTokens;
    private int queryOutputTokens;
    private int queryTotalTokens;
    private Map<String, Object> params;
    private String target;

    public NodeBuilder setNodeId(String nodeId) {
        this.nodeId = nodeId;
        return this;
    }

    public NodeBuilder setNodeType(String nodeType) {
        this.nodeType = nodeType;
        return this;
    }

    public NodeBuilder setNodeName(String nodeName) {
        this.nodeName = nodeName;
        return this;
    }

    public NodeBuilder setNodeInput(String nodeInput) {
        this.nodeInput = nodeInput;
        return this;
    }

    public NodeBuilder setNodeOutput(String nodeOutput) {
        this.nodeOutput = nodeOutput;
        return this;
    }

    public NodeBuilder setChainRootId(String chainRootId) {
        this.chainRootId = chainRootId;
        return this;
    }

    public NodeBuilder setChainId(String chainId) {
        this.chainId = chainId;
        return this;
    }

    public NodeBuilder setStep(int step) {
        this.step = step;
        return this;
    }

    public NodeBuilder setHasChildren(boolean hasChildren) {
        this.hasChildren = hasChildren;
        return this;
    }

    public NodeBuilder setInputs(Map<String, Object> inputs) {
        this.inputs = inputs;
        return this;
    }

    public NodeBuilder setPrompt(String prompt) {
        this.prompt = prompt;
        return this;
    }

    public NodeBuilder setResponse(String response) {
        this.response = response;
        return this;
    }

    public NodeBuilder setCreationTimestamp(long creationTimestamp) {
        this.creationTimestamp = creationTimestamp;
        return this;
    }

    public NodeBuilder setFinishReason(String finishReason) {
        this.finishReason = finishReason;
        return this;
    }

    public NodeBuilder setLatency(long latency) {
        this.latency = latency;
        return this;
    }

    public NodeBuilder setQueryInputTokens(int queryInputTokens) {
        this.queryInputTokens = queryInputTokens;
        return this;
    }

    public NodeBuilder setQueryOutputTokens(int queryOutputTokens) {
        this.queryOutputTokens = queryOutputTokens;
        return this;
    }

    public NodeBuilder setQueryTotalTokens(int queryTotalTokens) {
        this.queryTotalTokens = queryTotalTokens;
        return this;
    }

    public NodeBuilder setParams(Map<String, Object> params) {
        this.params = params;
        return this;
    }

    public NodeBuilder setTarget(String target) {
        this.target = target;
        return this;
    }

    public Node build() {
        Node node = new Node();
        node.setNode_id(nodeId);
        node.setNode_type(nodeType);
        node.setNode_name(nodeName);
        node.setNode_input(nodeInput);
        node.setNode_output(nodeOutput);
        node.setChain_root_id(chainRootId);
        node.setChain_id(chainId);
        node.setStep(step);
        node.setHas_children(hasChildren);
        node.setInputs(inputs);
        node.setPrompt(prompt);
        node.setResponse(response);
        node.setCreation_timestamp(creationTimestamp);
        node.setFinish_reason(finishReason);
        node.setLatency(latency);
        node.setQuery_input_tokens(queryInputTokens);
        node.setQuery_output_tokens(queryOutputTokens);
        node.setQuery_total_tokens(queryTotalTokens);
        node.setParams(params);
        node.setTarget(target);
        return node;
    }
}
