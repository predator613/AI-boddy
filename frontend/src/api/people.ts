import { apiRequest } from './client'
import { Contacts } from '../schemas/people'

/** GET /api/me/contacts – наставник и руководитель пользователя, отделы компании. */
export function getContacts() {
  return apiRequest('/api/me/contacts', Contacts)
}
