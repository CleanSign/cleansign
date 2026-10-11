// macOS Vision으로 이미지의 한국어 텍스트를 읽어 줄 단위로 출력한다.
// 실행: swift scripts/data/ocr_vision.swift <이미지...>   (줄은 위에서 아래, 왼쪽에서 오른쪽 순서)
import Foundation
import Vision
import AppKit

for path in CommandLine.arguments.dropFirst() {
    guard let img = NSImage(contentsOfFile: path),
          let cg = img.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
        FileHandle.standardError.write("읽을 수 없음: \(path)\n".data(using: .utf8)!); continue
    }
    let req = VNRecognizeTextRequest()
    req.recognitionLevel = .accurate
    req.recognitionLanguages = ["ko-KR", "en-US"]
    req.usesLanguageCorrection = true
    try VNImageRequestHandler(cgImage: cg).perform([req])
    let obs = (req.results ?? []).sorted {
        let dy = $0.boundingBox.midY - $1.boundingBox.midY
        return abs(dy) > 0.008 ? dy > 0 : $0.boundingBox.minX < $1.boundingBox.minX
    }
    print("=== \(URL(fileURLWithPath: path).lastPathComponent)")
    for o in obs {
        guard let c = o.topCandidates(1).first else { continue }
        print(String(format: "%.2f\t%@", c.confidence, c.string))
    }
}
