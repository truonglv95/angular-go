# Core Compiler Test Case Porting Audit (Global Matching)

- **Total TypeScript Spec Files**: 53
- **Total TypeScript Test Cases**: 1656
- **Total Globally Ported Test Cases**: 1102 (66.5%)
- **Total Go Test Files**: 54

| TS Spec File | TS Cases | Go Test Files | Go Cases | Port Status |
| --- | --- | --- | --- | --- |
| `compiler_facade_interface_spec.ts` | 0 | `A` | 0 | N/A (empty spec) |
| `expression_parser/ast_spec.ts` | 1 | `ast_test.go` | 1 | Fully Ported (1:1) |
| `expression_parser/lexer_spec.ts` | 93 | `lexer_test.go` | 91 | Partially Ported (91/93) |
| `expression_parser/parser_spec.ts` | 217 | `parser_test.go` | 178 | Partially Ported (178/217) |
| `expression_parser/serializer_spec.ts` | 25 | `serializer_test.go` | 25 | Fully Ported (1:1) |
| `i18n/digest_spec.ts` | 8 | `digest_test.go` | 7 | Partially Ported (7/8) |
| `i18n/extractor_merger_spec.ts` | 59 | `extractor_merger_test.go` | 58 | Partially Ported (58/59) |
| `i18n/i18n_ast_spec.ts` | 7 | `i18n_ast_test.go` | 7 | Fully Ported (1:1) |
| `i18n/i18n_html_parser_spec.ts` | 1 | `i18n_html_parser_test.go` | 1 | Fully Ported (1:1) |
| `i18n/i18n_parser_spec.ts` | 31 | `extractor_merger_test.go`, `i18n_parser_test.go` | 31 | Fully Ported (1:1) |
| `i18n/integration_xliff2_spec.ts` | 4 | `A` | 0 | Missing |
| `i18n/integration_xliff_spec.ts` | 4 | `A` | 0 | Missing |
| `i18n/integration_xmb_xtb_spec.ts` | 4 | `A` | 0 | Missing |
| `i18n/message_bundle_spec.ts` | 2 | `message_bundle_test.go` | 2 | Fully Ported (1:1) |
| `i18n/serializers/i18n_ast_spec.ts` | 2 | `i18n_ast_clone_recurse_test.go` | 2 | Fully Ported (1:1) |
| `i18n/serializers/placeholder_spec.ts` | 13 | `placeholder_registry_test.go` | 13 | Fully Ported (1:1) |
| `i18n/serializers/xliff2_spec.ts` | 10 | `xtb_test.go`, `xliff2_test.go` | 10 | Fully Ported (1:1) |
| `i18n/serializers/xliff_spec.ts` | 10 | `xtb_test.go`, `xliff2_test.go`, `xliff_test.go` | 10 | Fully Ported (1:1) |
| `i18n/serializers/xmb_spec.ts` | 2 | `xmb_test.go` | 2 | Fully Ported (1:1) |
| `i18n/serializers/xml_helper_spec.ts` | 7 | `xml_helper_test.go` | 7 | Fully Ported (1:1) |
| `i18n/serializers/xtb_spec.ts` | 14 | `xtb_test.go` | 13 | Partially Ported (13/14) |
| `i18n/translation_bundle_spec.ts` | 11 | `A` | 0 | Missing |
| `i18n/whitespace_sensitivity_spec.ts` | 4 | `whitespace_sensitivity_test.go` | 4 | Fully Ported (1:1) |
| `integration_spec.ts` | 2 | `A` | 0 | Missing |
| `ml_parser/ast_serializer_spec.ts` | 6 | `ast_serializer_test.go` | 6 | Fully Ported (1:1) |
| `ml_parser/html_parser_spec.ts` | 145 | `i18n_ast_clone_recurse_test.go`, `r3_template_transform_test.go`, `i18n_parser_test.go`, `lexer_test.go`, `html_parser_test.go` | 40 | Partially Ported (40/145) |
| `ml_parser/html_whitespaces_spec.ts` | 16 | `html_whitespaces_test.go` | 15 | Partially Ported (15/16) |
| `ml_parser/inline_comment_spec.ts` | 9 | `inline_comment_test.go` | 9 | Fully Ported (1:1) |
| `ml_parser/lexer_spec.ts` | 252 | `lexer_test.go`, `r3_template_transform_test.go`, `html_parser_test.go` | 243 | Partially Ported (243/252) |
| `output/abstract_emitter_node_only_spec.ts` | 8 | `emitter_visitor_context_test.go` | 7 | Partially Ported (7/8) |
| `output/abstract_emitter_spec.ts` | 6 | `abstract_emitter_test.go` | 6 | Fully Ported (1:1) |
| `output/output_jit_spec.ts` | 3 | `A` | 0 | Missing |
| `output/source_map_spec.ts` | 9 | `source_map_test.go` | 7 | Partially Ported (7/9) |
| `render3/r3_ast_absolute_span_spec.ts` | 56 | `r3_ast_spans_test.go`, `r3_ast_absolute_span_test.go` | 12 | Partially Ported (12/56) |
| `render3/r3_ast_spans_spec.ts` | 55 | `r3_ast_spans_test.go` | 55 | Fully Ported (1:1) |
| `render3/r3_ast_visitor_spec.ts` | 1 | `r3_ast_visitor_test.go` | 1 | Fully Ported (1:1) |
| `render3/r3_template_transform_spec.ts` | 256 | `lexer_test.go`, `r3_template_transform_test.go` | 80 | Partially Ported (80/256) |
| `render3/style_parser_spec.ts` | 13 | `parse_extracted_styles_test.go` | 11 | Partially Ported (11/13) |
| `render3/view/binding_spec.ts` | 70 | `t2_binder_test.go` | 16 | Partially Ported (16/70) |
| `render3/view/i18n_spec.ts` | 36 | `i18n_test.go` | 32 | Partially Ported (32/36) |
| `render3/view/parse_template_options_spec.ts` | 1 | `parse_template_options_test.go` | 1 | Fully Ported (1:1) |
| `schema/dom_element_schema_registry_spec.ts` | 30 | `dom_element_schema_registry_test.go` | 16 | Partially Ported (16/30) |
| `schema/trusted_types_sinks_spec.ts` | 3 | `trusted_types_sinks_test.go` | 3 | Fully Ported (1:1) |
| `selector/selector_spec.ts` | 34 | `selector_test.go` | 26 | Partially Ported (26/34) |
| `shadow_css/at_rules_spec.ts` | 19 | `A` | 0 | Missing |
| `shadow_css/host_and_host_context_spec.ts` | 24 | `shadow_css_test.go` | 23 | Partially Ported (23/24) |
| `shadow_css/keyframes_spec.ts` | 25 | `shadow_css_test.go` | 10 | Partially Ported (10/25) |
| `shadow_css/ng_deep_spec.ts` | 3 | `shadow_css_test.go` | 3 | Fully Ported (1:1) |
| `shadow_css/process_rules_spec.ts` | 7 | `A` | 0 | Missing |
| `shadow_css/repeat_groups_spec.ts` | 3 | `shadow_css_test.go` | 3 | Fully Ported (1:1) |
| `shadow_css/shadow_css_spec.ts` | 23 | `shadow_css_test.go` | 15 | Partially Ported (15/23) |
| `style_url_resolver_spec.ts` | 5 | `A` | 0 | Missing |
| `util_spec.ts` | 7 | `A` | 0 | Missing |