package api;

import lombok.Data;

import java.util.List;

@Data
public class PromptRegisteredScorersConfiguration {
    private String id;
    private String name;
    private List<String> aggregate_score_names;
    private String score_type;
    private List<String> scorable_node_types;
}
