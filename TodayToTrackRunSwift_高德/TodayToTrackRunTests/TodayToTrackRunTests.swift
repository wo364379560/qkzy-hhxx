//
//  TodayToTrackRunTests.swift
//  TodayToTrackRunTests
//
//  Created by HODGES FOWLER on 2026/7/12.
//

import XCTest
@testable import TodayToTrackRun

@MainActor
final class TodayToTrackRunTests: XCTestCase {

    override func setUpWithError() throws {
        // Put setup code here. This method is called before the invocation of each test method in the class.
    }

    override func tearDownWithError() throws {
        // Put teardown code here. This method is called after the invocation of each test method in the class.
    }

    func testExample() throws {
        // This is an example of a functional test case.
        // Use XCTAssert and related functions to verify your tests produce the correct results.
        // Any test you write for XCTest can be annotated as throws and async.
        // Mark your test throws to produce an unexpected failure when your test encounters an uncaught error.
        // Mark your test async to allow awaiting for asynchronous code to complete. Check the results with assertions afterwards.
    }

    func testLongChatTextIsKeptWholeAndUsesUnlimitedLines() {
        let source = String(repeating: "这是一段需要完整显示的中文消息1234567890", count: 8)
        let cell = LinLanChatMessageCell(style: .default, reuseIdentifier: nil)

        cell.configureTextForLayoutTest(source)

        XCTAssertEqual(cell.displayedBodyTextForTesting, source)
        XCTAssertTrue(cell.bodyUsesUnlimitedCharacterWrappingForTesting)
        XCTAssertFalse(cell.displayedBodyTextForTesting.contains("…"))
    }

    func testKickedTipVisibilityMatchesViewerRole() {
        XCTAssertTrue(LinLanImManager.shouldHideGroupTipType(
            .V2TIM_GROUP_TIPS_TYPE_KICKED,
            viewerRole: nil
        ))
        XCTAssertTrue(LinLanImManager.shouldHideGroupTipType(
            .V2TIM_GROUP_TIPS_TYPE_KICKED,
            viewerRole: LinLanImManager.roleMember
        ))
        XCTAssertFalse(LinLanImManager.shouldHideGroupTipType(
            .V2TIM_GROUP_TIPS_TYPE_KICKED,
            viewerRole: LinLanImManager.roleOwner
        ))
        XCTAssertFalse(LinLanImManager.shouldHideGroupTipType(
            .V2TIM_GROUP_TIPS_TYPE_KICKED,
            viewerRole: LinLanImManager.roleAdmin
        ))
        XCTAssertFalse(LinLanImManager.shouldHideGroupTipType(
            .V2TIM_GROUP_TIPS_TYPE_JOIN,
            viewerRole: nil
        ))
    }

    func testCompleteMembershipSnapshotKeepsCommunityTopics() {
        let parent = "@TGS#_community123"
        let topic = parent + "@TOPIC#_running"
        let visible = LinLanGroupMembershipPolicy.visibleGroupIDs(
            conversationGroupIDs: [topic, "other"],
            joinedGroupIDs: [parent],
            removedGroupIDs: []
        )

        XCTAssertEqual(visible, [topic])
    }

    func testIncompleteMembershipSnapshotOnlyFiltersKnownRemovedGroupAndTopics() {
        let parent = "@TGS#_community123"
        let topic = parent + "@TOPIC#_running"
        let visible = LinLanGroupMembershipPolicy.visibleGroupIDs(
            conversationGroupIDs: [parent, topic, "kept"],
            joinedGroupIDs: nil,
            removedGroupIDs: [parent]
        )

        XCTAssertEqual(visible, ["kept"])
    }

    func testRemovedGroupStateIsIsolatedAndLegalRejoinRestoresIt() throws {
        let suite = "TodayToTrackRunTests.membership.\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suite))
        defer { defaults.removePersistentDomain(forName: suite) }
        let group = "group-1"

        LinLanGroupMembershipState.markRemoved(
            groupIDs: [group],
            scope: "sdk:account-a",
            defaults: defaults
        )
        XCTAssertTrue(LinLanGroupMembershipState.isRemoved(
            groupID: group,
            scope: "sdk:account-a",
            defaults: defaults
        ))
        XCTAssertFalse(LinLanGroupMembershipState.isRemoved(
            groupID: group,
            scope: "sdk:account-b",
            defaults: defaults
        ))

        LinLanGroupMembershipState.reconcileJoined(
            groupIDs: [group],
            scope: "sdk:account-a",
            defaults: defaults
        )
        XCTAssertFalse(LinLanGroupMembershipState.isRemoved(
            groupID: group,
            scope: "sdk:account-a",
            defaults: defaults
        ))
    }

    func testPerformanceExample() throws {
        // This is an example of a performance test case.
        self.measure {
            // Put the code you want to measure the time of here.
        }
    }

}
