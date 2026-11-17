declare module "trtc-cloud-js-sdk" {
  export enum TRTCAppScene {
    TRTCAppSceneVideoCall = 0,
    TRTCAppSceneAudioCall = 2,
  }

  export enum TRTCVideoStreamType {
    TRTCVideoStreamTypeBig = 0,
  }

  export enum TRTCAudioQuality {
    TRTCAudioQualitySpeech = 1,
  }

  export enum TRTCRoleType {
    TRTCRoleAnchor = 20,
  }

  export class TRTCParams {
    constructor(
      sdkAppId?: number,
      userId?: string,
      userSig?: string,
      roomId?: number,
      strRoomId?: string,
      role?: TRTCRoleType,
    );
    sdkAppId: number;
    userId: string;
    userSig: string;
    roomId: number;
    strRoomId: string;
    role: TRTCRoleType;
  }

  export default class TRTCCloud {
    static getTRTCShareInstance(options?: any): TRTCCloud;
    static destroyTRTCShareInstance(): void;
    on(eventName: string, handler: (...args: any[]) => void): void;
    off(eventName: string, handler: (...args: any[]) => void): void;
    enterRoom(params: TRTCParams | Record<string, any>, scene: TRTCAppScene): Promise<any> | void;
    exitRoom(): Promise<any> | void;
    startLocalAudio(quality?: TRTCAudioQuality): Promise<any> | void;
    stopLocalAudio(): Promise<any> | void;
    muteLocalAudio(muted: boolean): Promise<any> | void;
    startLocalPreview(view?: string | HTMLElement): Promise<any> | void;
    stopLocalPreview(): Promise<any> | void;
    startRemoteView(userID: string, view?: string | HTMLElement, streamType?: TRTCVideoStreamType): Promise<any> | void;
    stopRemoteView(userID: string, streamType?: TRTCVideoStreamType): Promise<any> | void;
  }
}
