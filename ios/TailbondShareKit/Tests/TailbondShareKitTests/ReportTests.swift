import Foundation
import XCTest
@testable import TailbondShareKit

/// Report a problem: the port against the Go reference's vectors (agent/internal/report).
final class ReportTests: XCTestCase {
    func vectors() throws -> [String: Any] {
        var root = URL(fileURLWithPath: #filePath)
        for _ in 0..<5 { root.deleteLastPathComponent() }
        let data = try Data(contentsOf: root.appendingPathComponent("agent/internal/report/testdata/vectors.json"))
        return try JSONSerialization.jsonObject(with: data) as! [String: Any]
    }

    func testRedactsLikeGo() throws {
        let cases = try vectors()["redact"] as! [[String: Any]]
        XCTAssertGreaterThanOrEqual(cases.count, 20)
        for c in cases {
            XCTAssertEqual(Redactor.redact(c["in"] as! String), c["out"] as? String, c["name"] as! String)
        }
    }

    func testEscapesLikeGo() throws {
        for c in try vectors()["escape"] as! [[String: Any]] {
            XCTAssertEqual(ProblemReport.escape(c["in"] as! String), c["out"] as? String)
        }
    }

    func testReportsLikeGo() throws {
        let cases = try vectors()["reports"] as! [[String: Any]]
        XCTAssertGreaterThanOrEqual(cases.count, 4)
        for c in cases {
            let r = c["report"] as! [String: String]
            let rep = ProblemReport(component: r["component"]!, version: r["version"]!, device: r["device"]!, what: r["what"]!,
                                    steps: r["steps"]!, logs: r["logs"]!, crash: r["crash"]!)
            let name = c["name"] as! String
            XCTAssertEqual(rep.full(), c["full"] as? String, name + " full")
            XCTAssertEqual(rep.issueURL(), c["url"] as? String, name + " url")
            XCTAssertLessThanOrEqual(rep.issueURL().utf8.count, ReportText.maxURL)
        }
    }

    func testSameWords() throws {
        let t = try vectors()["texts"] as! [String: String]
        XCTAssertEqual(t["notice"], ReportText.notice)
        XCTAssertEqual(t["confirm"], ReportText.confirm)
        XCTAssertEqual(t["security_note"], ReportText.securityNote)
        XCTAssertEqual(t["security_url"], ReportText.securityURL)
        XCTAssertEqual(t["repo"], ReportText.repo)
        XCTAssertEqual(t["max_url"], String(ReportText.maxURL))
        XCTAssertEqual(ProblemReport.fileName("2026-11-04T06:51:24Z"), "tailbond-report-20261104-065124.txt")
    }
}
