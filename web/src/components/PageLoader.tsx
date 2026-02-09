import { motion } from '@/lib/motion'

export function PageLoader() {
  return (
    <div className="space-y-6 p-2">
      <div className="space-y-3">
        <motion.div
          className="h-8 w-48 rounded-lg loading-shimmer"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0 }}
        />
        <motion.div
          className="h-4 w-72 rounded-md loading-shimmer"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.1 }}
        />
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <motion.div
            key={i}
            className="h-28 rounded-xl loading-shimmer"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 + i * 0.08, type: 'spring', bounce: 0.2, duration: 0.5 }}
          />
        ))}
      </div>
      <motion.div
        className="h-64 rounded-xl loading-shimmer"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4, type: 'spring', bounce: 0.2, duration: 0.5 }}
      />
    </div>
  )
}

export function InlineLoader() {
  return (
    <div className="flex h-64 items-center justify-center">
      <motion.div
        className="flex items-center gap-3 text-muted-foreground"
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ type: 'spring', bounce: 0.3, duration: 0.5 }}
      >
        <motion.div
          className="h-5 w-5 rounded-full border-2 border-primary/30 border-t-primary"
          animate={{ rotate: 360 }}
          transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }}
        />
        <span className="text-sm font-medium">加载中...</span>
      </motion.div>
    </div>
  )
}
