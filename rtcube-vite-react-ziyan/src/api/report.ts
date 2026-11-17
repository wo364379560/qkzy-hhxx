import { chatPost } from './chatHttp'

export type ReportCategory = 'spam' | 'abuse' | 'fraud' | 'other'

export async function submitReport(payload: {
  targetId: string
  targetType: 'user' | 'group'
  targetName: string
  category: ReportCategory
  description?: string
}) {
  await chatPost(
    '/reports',
    {
      target_id: payload.targetId,
      target_type: payload.targetType,
      target_name: payload.targetName,
      category: payload.category,
      description: payload.description,
    },
    true,
  )
}
