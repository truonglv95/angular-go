# Kế Hoạch Tối Ưu Hóa Hiệu Năng Cấp Thấp (High-Performance Architecture Plan)

Tài liệu này ghi chú các hạng mục kiến trúc cần đo và tối ưu để biến `go-ngc` từ một bản port 1:1 của TypeScript/NodeJS thành một compiler native thực thụ. Kỳ vọng 20x-50x chỉ nên xem là mục tiêu dài hạn cho các phase rất hẹp như linker hoặc syntax-directed local compilation; builder end-to-end hiện tại còn bị chi phối bởi TypeScript program setup, emit, Vite optimizeDeps, esbuild/Rollup, và kích thước dependency graph.

## Số đo hiện tại

Command:

```bash
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH node angular-packages/compiler_cli/tools/bench/run_benchmarks.mjs
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH NG_BENCH_COMPONENTS=2000 node angular-packages/compiler_cli/tools/bench/generate_large_app.mjs
```

Kết quả mới nhất:

- Demo app cold compile: Go `395.77ms`, ngtsc `1561.01ms`, speedup `3.94x`.
- Demo app linker `primeng-button`: Go `25.91ms`, ngtsc linker/Babel `327.85ms`, speedup `12.65x`.
- Demo app Vite dev startup: `229.67ms`.
- Demo app Vite production build: `4473.34ms`; phần này không scale theo compiler-only speed vì bundling và dependency optimization chiếm phần lớn.
- Synthetic 500 components: Go global `305.53ms`, Go local `304.83ms`, ngtsc `2947.74ms`, speedup khoảng `9.6x`.
- Synthetic 2000 components: Go global `1051.06ms`, Go local `934.59ms`, ngtsc `6310.12ms`, speedup `6.00x` global và `6.75x` local.

Kết luận hiện tại:

- Local compilation/parallel analyze mới chỉ cải thiện nhẹ ở synthetic 2000 components, khoảng 12% so với global mode.
- Bottleneck chưa nằm chủ yếu ở decorator analyze, nên chỉ thêm goroutine quanh analyze không thể tạo 20x.
- CPU profile của synthetic 2000 components cho thấy runtime/GC/allocation và file/syscall overhead chiếm đáng kể; template parse/compile và emit vẫn tuần tự đủ nhiều để giới hạn speedup.
- Builder end-to-end cần tối ưu riêng Vite/linker/cache, không thể suy ra từ cold compile speedup.

## 1. Data-Oriented Design (DOD) & Memory Arenas
**Vấn đề hiện tại:** Trình biên dịch đang cấp phát hàng triệu đối tượng con trỏ (`*ast.Node`, `*ast.Symbol`) trên Heap. GC của Go phải tốn tới 80% thời gian CPU để theo dõi và dọn dẹp các con trỏ ngắn hạn này.
**Hành động cần làm:**
- Thay thế toàn bộ hệ thống cấp phát đối tượng AST sang sử dụng **Memory Arenas** (bể nhớ khối).
- Khai báo các đối tượng AST dưới dạng cấu trúc phẳng (Flat structs) bên trong các Mảng khổng lồ (Arrays).
- Thay thế mọi thuộc tính dùng con trỏ (ví dụ: `Parent *ast.Node`) bằng các chỉ mục số nguyên 32-bit (ví dụ: `ParentID uint32`).
- *Lợi ích:* Xóa bỏ hoàn toàn gánh nặng cho Garbage Collector. Xóa bỏ node chỉ bằng cách reset mảng (Mất 1ms).

## 2. Multi-threading & Parallelism (Xử lý đa luồng)
**Vấn đề hiện tại:** Hệ thống đang biên dịch đơn luồng (Single-threaded), không tận dụng được sức mạnh của CPU đa nhân.
**Hành động cần làm:**
- **Parallel Parsing:** Sử dụng `goroutines` để phân tích cú pháp (Parse) hàng nghìn file TypeScript và HTML (`.ts`, `.html`) cùng lúc.
- **Concurrent Dependency Resolution:** Xây dựng Đồ thị phụ thuộc (DAG - Directed Acyclic Graph) để tìm ra các module/component không phụ thuộc lẫn nhau, sau đó đẩy chúng vào Worker Pool để phân tích (Analyze) và biên dịch (Emit) song song.

## 3. String Interning (Bể chứa chuỗi tập trung)
**Vấn đề hiện tại:** Trình biên dịch lãng phí tài nguyên để tạo và so sánh vô số chuỗi lặp đi lặp lại (`Component`, `Input`, tên biến, tên hàm, ...).
**Hành động cần làm:**
- Xây dựng một **String Pool (Symbol Table)** toàn cục.
- Mỗi chuỗi string khi được sinh ra sẽ lưu vào Pool và nhận về một ID nguyên (Integer ID).
- Ở tất cả các vòng lặp nội bộ (Type-Checking, Lookup), thay vì so sánh 2 chuỗi (`string == string`), chỉ cần so sánh 2 số nguyên (`ID1 == ID2`).
- *Lợi ích:* Giảm thiểu chi phí bộ nhớ đệm và tăng tốc độ so sánh (Lookup) lên nhiều lần.

## 4. Lazy Evaluation (Kiểm tra kiểu lười biếng)
**Vấn đề hiện tại:** Trình biên dịch (Đặc biệt là Type Checker của `typescript-go`) đang quét và Resolve tất cả các Type một cách tham lam (Eagerly).
## 4. Tối ưu hóa Evaluation (Lazy vs Eager)
- **Vấn đề:** Angular AoT compiler đôi khi "evaluate" (đánh giá) trước toàn bộ cây phụ thuộc dù không sử dụng hết, gây tốn CPU vô ích.
- **Giải pháp:** Áp dụng chặt chẽ "Thunk" (Hàm trả về kết quả bị trì hoãn) cho mọi quá trình Type checking và Trait Resolution.

## 5. Tối ưu hóa Tối thượng: Local Compilation / Syntax-directed AoT
- **Vấn đề:** Hiện tại `go-ngc` bị khóa trong kiến trúc đơn luồng do phụ thuộc vào TypeScript Type Checker (hệ thống `LinkStore` không an toàn cho đa luồng) để truy xuất các Metadata của Angular (như Dependency Injection).
- **Giải pháp (Lấy cảm hứng từ Rust/SWC):**
  - Loại bỏ hoàn toàn sự phụ thuộc vào Type Checker trong bước phân tích AoT (`AnalyzeSync`).
  - Viết lại `ReflectionHost` để chỉ đọc đồ thị Cú pháp (AST) thay vì Đồ thị Ngữ nghĩa (Semantic). Ví dụ: Khi xử lý Injection, chỉ cần tìm chuỗi Import tương ứng trên đầu file (Syntax-directed).
  - Khi đã ngắt kết nối với Type Checker, chúng ta có thể áp dụng **Xử lý Đa luồng (goroutines)** cho toàn bộ quá trình parse và compile Angular, đẩy tốc độ giảm xuống mức tính bằng mili-giây (< 0.5s cho 5000 files).

---
**Lưu ý khi triển khai:**
Đây là một cuộc đại phẫu thuật (Massive Refactor). Thay đổi số `(1)` đòi hỏi chúng ta phải viết lại gần như toàn bộ cấu trúc dữ liệu của `typescript-go` (vốn đang có hàng chục ngàn dòng code phụ thuộc vào con trỏ). Cần lên kế hoạch triển khai từng bước một để tránh phá vỡ tính chính xác (Fail-normalized) của trình biên dịch gốc.
