 ## 📊 Trạng thái hiện tại của Port ngtsc → Go                                                                                                                                                             
                                                                                                                                                                                                            
  ### 🔴 Vấn đề Test ngay lập tức                                                                                                                                                                           
                                                                                                                                                                                                            
  Kết quả  go test ./angular-packages/... :                                                                                                                                                                 
                                                                                                                                                                                                            
   Package                                                                           │ Kết quả
  ───────────────────────────────────────────────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
    compiler/expression_parser                                                       │ ❌ Build failed —  parser_test.go  bị empty (chỉ có  \n , không có  package  declaration)
    compiler/ml_parser                                                               │ ❌ KILLED sau 54.6 giây — timeout/crash
    compiler_cli/reflection                                                          │ ❌ Build failed — test gọi  NewTypeScriptReflectionHost(nil, bool)  nhưng implementation chỉ nhận  (any) 
    compiler_cli/partial_evaluator                                                   │ ❌ Build failed —  NewPartialEvaluator  cần 3 args, test truyền 2
    compiler_cli/annotations/component                                               │ ❌ Build failed —  TypeScriptReflectionHost  không implement  ReflectionHost  (sai return type  GetExportsOfModule )
    compiler_cli/annotations/injectable                                              │ ❌ Build failed — comparison  Name != nil  always true
                                                                                                                                                                                                            
  Các package không có test (chỉ build):                                                                                                                                                                    
                                                                                                                                                                                                            
  •  compiler/chars ,  compiler/core ,  compiler/output ,  compiler/schema ,  compiler/tags , etc. —  [no test files]  ✅                                                                                   
  ──────                                                                                                                                                                                                    
  ### 🐛 Các Bug cần fix ngay                                                                                                                                                                               
                                                                                                                                                                                                            
  1.  parser_test.go  bị empty:                                                                                                                                                                             
                                                                                                                                                                                                            
    angular-packages/compiler/expression_parser/parser_test.go:1:1: expected 'package', found 'EOF'                                                                                                         
                                                                                                                                                                                                            
  File này hoàn toàn trống — cần thêm  package expression_parser  ít nhất.                                                                                                                                  
                                                                                                                                                                                                            
  2.  reflection.NewTypeScriptReflectionHost  signature mismatch:                                                                                                                                           
                                                                                                                                                                                                            
  • Test gọi:  NewTypeScriptReflectionHost(nil, false)  — 2 args                                                                                                                                            
  • Implementation:  func NewTypeScriptReflectionHost(checker any)  — 1 arg                                                                                                                                 
  • Interface  ReflectionHost  khai báo  GetExportsOfModule  trả  map[string]Declaration  nhưng implementation trả  map[string]any                                                                          
                                                                                                                                                                                                            
  3.  partial_evaluator.NewPartialEvaluator  thiếu arg:                                                                                                                                                     
                                                                                                                                                                                                            
  • Test gọi:  NewPartialEvaluator(host, nil)  — 2 args                                                                                                                                                     
  • Implementation cần: 3 args                                                                                                                                                                              
  ──────                                                                                                                                                                                                    
  ### 📈 Tổng quan Coverage Port (theo Gap Report)                                                                                                                                                          
                                                                                                                                                                                                            
  ####  compiler/  — Lõi biên dịch Angular (~44% tổng thể)                                                                                                                                                  
                                                                                                                                                                                                            
   Module                                           │ TS Lines                                         │ Go Lines                                         │ Coverage
  ──────────────────────────────────────────────────┼──────────────────────────────────────────────────┼──────────────────────────────────────────────────┼─────────────────────────────────────────────────
    expression_parser                               │ 3,903                                            │ 3,104                                            │ 79% 🟢 Deep Port
    ml_parser                                       │ 6,317                                            │ 4,648                                            │ 73% 🟡 Partial
    template/pipeline/phases                        │ 8,767                                            │ 6,808                                            │ 77% 🟢 Deep Port
    template/pipeline                               │ 12,692                                           │ 7,944                                            │ 62% 🟡 Partial
    output                                          │ 3,535                                            │ 2,537                                            │ 71% 🟡 Partial
    schema                                          │ 829                                              │ 554                                              │ 66% 🟡 Partial
    i18n                                            │ 4,109                                            │ 680                                              │ 16% 🔴
    render3  (root)                                 │ 12,843                                           │ 3,140                                            │ 24% 🔴
    render3/partial                                 │ 1,798                                            │ 0                                                │ 0% ❌ Missing
    render3/view                                    │ 4,929                                            │ 302                                              │ 6% 🔴
    template/pipeline/ir                            │ 6,047                                            │ 528                                              │ 8% 🔴
                                                                                                                                                                                                            
  ####  compiler_cli/ngtsc  — CLI (~thấp)                                                                                                                                                                   
                                                                                                                                                                                                            
   Module                                                                                              │ Coverage
  ─────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────
    partial_evaluator                                                                                  │ 56% 🟡
    reflection                                                                                         │ 15% 🔴
    annotations                                                                                        │ 11% 🔴
    cycles ,  diagnostics ,  metadata ,  scope ,  perf , etc.                                          │ 0% ❌
