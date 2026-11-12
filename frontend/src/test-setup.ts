import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Vitest runs without globals, so Testing Library can't register its own
// auto-cleanup; unmount after every test so tests don't see each other's DOM.
afterEach(cleanup)
