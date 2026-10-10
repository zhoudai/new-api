/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ModelRatioVisualEditor } from '../model-ratio-visual-editor'

let client: QueryClient | undefined
const originalAuth = useAuthStore.getState().auth

afterEach(() => {
  client?.clear()
  localStorage.clear()
  window.getSelection()?.removeAllRanges()
  useAuthStore.setState({ auth: originalAuth })
  vi.restoreAllMocks()
})

async function renderEditor(isMobile = false, taskModelNames: string[] = []) {
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 1, username: 'admin', role: ROLE.SUPER_ADMIN },
    },
  })
  if (isMobile) {
    const matchMedia = window.matchMedia.bind(window)
    vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
      ...matchMedia(query),
      matches: query === '(max-width: 767px)' || matchMedia(query).matches,
    }))
  }

  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/pricing') {
      return { data: { success: true, data: [], vendors: [] } }
    }
    if (url === '/api/option/model_pricing') {
      return {
        data: {
          success: true,
          data: {
            entries: taskModelNames.map((model_name) => ({
              model_name,
              usage_schema: { images: { type: 'number', unit: 'image' } },
            })),
            options: {},
            empty_version: 'test-version',
          },
        },
      }
    }
    return { data: { success: true, data: {} } }
  })
  const modelRatio = JSON.stringify({
    'gpt-4.1-mini': 0.2,
    'claude-sonnet': 1.5,
    ...Object.fromEntries(taskModelNames.map((name) => [name, 1])),
  })
  const taskBillingMode = JSON.stringify(
    Object.fromEntries(taskModelNames.map((name) => [name, 'tiered_expr']))
  )
  const taskBillingExpr = JSON.stringify(
    taskModelNames[0] ? { [taskModelNames[0]]: 'u("images")' } : {}
  )
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client = queryClient
  const view = render(
    <QueryClientProvider client={client}>
      <ModelRatioVisualEditor
        savedModelPrice='{}'
        savedModelRatio={modelRatio}
        savedCacheRatio='{}'
        savedCreateCacheRatio='{}'
        savedCompletionRatio='{}'
        savedImageRatio='{}'
        savedAudioRatio='{}'
        savedAudioCompletionRatio='{}'
        savedBillingMode={taskBillingMode}
        savedBillingExpr={taskBillingExpr}
        modelPrice='{}'
        modelRatio={modelRatio}
        cacheRatio='{}'
        createCacheRatio='{}'
        completionRatio='{}'
        imageRatio='{}'
        audioRatio='{}'
        audioCompletionRatio='{}'
        billingMode={taskBillingMode}
        billingExpr={taskBillingExpr}
        onChange={vi.fn()}
        onSave={vi.fn()}
        isSaving={false}
      />
    </QueryClientProvider>
  )
  await waitFor(() => expect(queryClient.isFetching()).toBe(0))
  return view
}

it.each([false, true])(
  'does not request the public pricing catalog from the admin editor (mobile=%s)',
  async (isMobile) => {
    await renderEditor(isMobile)
    const row = await screen.findByRole('row', { name: /gpt-4\.1-mini/ })
    fireEvent.click(within(row).getByText('gpt-4.1-mini'))
    await screen.findByRole('region', { name: 'Edit model pricing' })

    expect(api.get).not.toHaveBeenCalledWith('/api/pricing')
  }
)

it('keeps task pricing badges and filtering when the public pricing catalog is disabled', async () => {
  const user = userEvent.setup()
  await renderEditor(false, ['task-configured', 'task-unconfigured'])

  const configuredRow = await screen.findByRole('row', {
    name: /task-configured/,
  })
  const unconfiguredRow = await screen.findByRole('row', {
    name: /task-unconfigured/,
  })
  expect(within(configuredRow).getByText('Task pricing')).toBeVisible()
  expect(
    within(unconfiguredRow).getByText('Task pricing not configured')
  ).toBeVisible()

  const modeFilter = screen
    .getAllByRole('button', { name: 'Mode' })
    .find((button) => button.getAttribute('aria-haspopup') === 'dialog')
  if (!modeFilter) throw new Error('Expected the Mode filter button')
  await user.click(modeFilter)
  await user.click(screen.getByText('Expression - Task pricing'))

  expect(
    await screen.findByRole('row', { name: /task-configured/ })
  ).toBeInTheDocument()
  expect(
    screen.queryByRole('row', { name: /task-unconfigured/ })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('row', { name: /gpt-4\.1-mini/ })
  ).not.toBeInTheDocument()
  expect(api.get).not.toHaveBeenCalledWith('/api/pricing')
})

// A drag-select ends with mousedown/mouseup inside the row and the browser
// then fires `click` on the row. Only that final `click` is dispatched here so
// the text selection created by the drag is still active when the row opens
// the editor.
it('keeps the drag-selected model name highlighted when the row click opens the editor', async () => {
  await renderEditor()
  const row = await screen.findByRole('row', { name: /gpt-4\.1-mini/ })
  const nameCell = within(row).getByText('gpt-4.1-mini')
  window.getSelection()?.selectAllChildren(nameCell)
  expect(String(window.getSelection())).toBe('gpt-4.1-mini')

  fireEvent.click(nameCell)

  expect(
    await screen.findByRole('region', { name: 'Edit model pricing' })
  ).toBeInTheDocument()
  expect(nameCell).toBeInTheDocument()
  expect(String(window.getSelection())).toBe('gpt-4.1-mini')
})

it('keeps other rows mounted when a different model is opened for editing', async () => {
  await renderEditor()
  const targetRow = await screen.findByRole('row', { name: /gpt-4\.1-mini/ })
  const otherName = within(
    screen.getByRole('row', { name: /claude-sonnet/ })
  ).getByText('claude-sonnet')

  fireEvent.click(within(targetRow).getByText('gpt-4.1-mini'))

  expect(
    await screen.findByRole('region', { name: 'Edit model pricing' })
  ).toBeInTheDocument()
  expect(otherName).toBeInTheDocument()
})
