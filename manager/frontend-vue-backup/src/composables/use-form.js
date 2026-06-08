import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'

/**
 * Creates a vee-validate form bound to a zod schema.
 * Usage:
 *   const { handleSubmit, setValues } = createForm(myZodSchema, { field: '' })
 *   const onSubmit = handleSubmit(values => api.save(values))
 */
export function createForm(zodSchema, initialValues = {}) {
  const schema = toTypedSchema(zodSchema)
  return useForm({ validationSchema: schema, initialValues })
}
