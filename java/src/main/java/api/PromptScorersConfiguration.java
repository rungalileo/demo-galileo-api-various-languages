package api;

import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class PromptScorersConfiguration {
    private boolean latency;
    private boolean cost;
    private boolean pii;
    private boolean input_pii;
    private boolean bleu;
    private boolean rouge;
    private boolean protect_status;
    private boolean context_relevance;
    private boolean toxicity;
    private boolean input_toxicity;
    private boolean tone;
    private boolean input_tone;
    private boolean sexist;
    private boolean input_sexist;
    private boolean prompt_injection;
    private boolean adherence_nli;
    private boolean chunk_attribution_utilization_nli;
    private boolean completeness_nli;
    private boolean uncertainty;
    private boolean factuality;
    private boolean groundedness;
    private boolean prompt_perplexity;
    private boolean chunk_attribution_utilization_gpt;
    private boolean completeness_gpt;
}
