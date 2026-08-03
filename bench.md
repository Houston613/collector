

# Бенчмарки
Были добавлены бенчмарки для тестирования скорости и аллокаций:
- Для репозитория ([storage_bench_test.go](file:///C:/Users/houston/projects/collector/internal/repository/storage_bench_test.go)): `BenchmarkUpdateGauge`, `BenchmarkUpdateCounter`, `BenchmarkUpdateMetrics`.
- Для подписи ([signature_bench_test.go](file:///C:/Users/houston/projects/collector/pkg/signature/signature_bench_test.go)): `BenchmarkSign`, `BenchmarkVerify`.
- Для сжатия ответов ([gzip_bench_test.go](file:///C:/Users/houston/projects/collector/internal/middleware/gzip_bench_test.go)): `BenchmarkGzipMiddleware`.

# Оптимизации GzipMiddleware
Было обнаружено, что `GzipMiddleware` при каждом запросе создавал новый gzip writer (`gzip.NewWriterLevel`), что вызывало большие объёмы аллокаций памяти (`compress/flate.NewWriter` составлял около 80% всех аллокаций памяти).

Для оптимизации был внедрён пул `sync.Pool` для переиспользования `gzip.Writer` с помощью метода `Reset`.

Сравнение потребления памяти до и после оптимизации с помощью команды:
```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Результат сравнения (отрицательные значения показывают уменьшение аллокаций):
```text
PS C:\Users\houston\projects\collector> go tool pprof -top "-diff_base=profiles/base.pprof" profiles/result.pprof
File: middleware.test.exe
Build ID: C:\Users\houston\AppData\Local\Temp\go-build4207179133\b001\middleware.test.exe2026-07-03 16:46:06.0151111 +0300 MSK
Type: alloc_space
Time: 2026-07-03 16:45:08 MSK
Showing nodes accounting for -6267.62MB, 95.56% of 6558.82MB total
Dropped 60 nodes (cum <= 32.79MB)
      flat  flat%   sum%        cum   cum%
-5211.91MB 79.46% 79.46% -6377.09MB 97.23%  compress/flate.NewWriter (inline)
-1124.15MB 17.14% 96.60% -1124.15MB 17.14%  compress/flate.(*compressor).initDeflate (inline)
  160.62MB  2.45% 94.15%   160.62MB  2.45%  bufio.NewReaderSize (inline)
  -71.66MB  1.09% 95.25%   -71.66MB  1.09%  compress/flate.(*huffmanEncoder).generate
  -22.52MB  0.34% 95.59%   -41.03MB  0.63%  compress/flate.newHuffmanBitWriter (inline)
    2.50MB 0.038% 95.55% -6413.12MB 97.78%  collector/internal/middleware.BenchmarkGzipMiddleware.GzipMiddleware.func2.1
   -0.50MB 0.0076% 95.56% -6358.47MB 96.95%  github.com/labstack/echo/v4.(*context).String
         0     0% 95.56%   160.62MB  2.45%  bufio.NewReader (inline)
         0     0% 95.56%   -66.16MB  1.01%  collector/internal/middleware.(*compressWriter).Close
         0     0% 95.56% -6375.09MB 97.20%  collector/internal/middleware.(*compressWriter).Write
         0     0% 95.56% -6213.50MB 94.74%  collector/internal/middleware.BenchmarkGzipMiddleware
         0     0% 95.56%   -66.16MB  1.01%  collector/internal/middleware.BenchmarkGzipMiddleware.GzipMiddleware.func2.1.1
         0     0% 95.56% -6358.47MB 96.95%  collector/internal/middleware.BenchmarkGzipMiddleware.func1
         0     0% 95.56%   -66.16MB  1.01%  compress/flate.(*Writer).Close (inline)
         0     0% 95.56%   -66.16MB  1.01%  compress/flate.(*compressor).close
         0     0% 95.56%   -71.66MB  1.09%  compress/flate.(*compressor).deflate
         0     0% 95.56% -1165.19MB 17.77%  compress/flate.(*compressor).init
         0     0% 95.56%   -71.66MB  1.09%  compress/flate.(*compressor).writeBlock
         0     0% 95.56%   -43.09MB  0.66%  compress/flate.(*huffmanBitWriter).indexTokens
         0     0% 95.56%   -71.66MB  1.09%  compress/flate.(*huffmanBitWriter).writeBlock
         0     0% 95.56%   -66.16MB  1.01%  compress/gzip.(*Writer).Close
         0     0% 95.56% -6375.09MB 97.20%  compress/gzip.(*Writer).Write
         0     0% 95.56% -6413.24MB 97.78%  github.com/labstack/echo/v4.(*Echo).ServeHTTP
         0     0% 95.56% -6357.59MB 96.93%  github.com/labstack/echo/v4.(*Echo).add.func1
         0     0% 95.56% -6375.09MB 97.20%  github.com/labstack/echo/v4.(*Response).Write
         0     0% 95.56% -6357.97MB 96.94%  github.com/labstack/echo/v4.(*context).Blob
         0     0% 95.56%   181.12MB  2.76%  net/http/httptest.NewRequest (inline)
         0     0% 95.56%   181.12MB  2.76%  net/http/httptest.NewRequestWithContext
         0     0% 95.56% -6213.47MB 94.73%  testing.(*B).launch
         0     0% 95.56%    -6213MB 94.73%  testing.(*B).runN
```

В результате использования пула количество аллокаций памяти снизилось на **~6.3 ГБ** за время теста.
