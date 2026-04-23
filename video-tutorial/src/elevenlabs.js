/**
 * ElevenLabs TTS integration — build-time module.
 *
 * Synthesizes narration text into audio with word-level timestamps.
 * Uses the /v1/text-to-speech/{voice_id}/with-timestamps endpoint.
 *
 * Returns: { audioBase64, wordTimestamps: [{ word, start, end }] }
 */

const https = require('https');

const ELEVENLABS_BASE = 'api.elevenlabs.io';
const DEFAULT_MODEL = 'eleven_multilingual_v2';
const RETRY_MAX = 2;
const RETRY_DELAY_MS = 1000;
const SEGMENT_GAP_MS = 200;

/**
 * Convert character-level alignment data to word-level timestamps.
 * Groups consecutive non-space characters into words.
 *
 * @param {Object} alignment - { characters: string[], character_start_times_seconds: number[], character_end_times_seconds: number[] }
 * @returns {Array<{ word: string, start: number, end: number }>}
 */
function characterTimestampsToWords(alignment) {
  const { characters, character_start_times_seconds: starts, character_end_times_seconds: ends } = alignment;
  const words = [];
  let currentWord = '';
  let wordStart = -1;
  let wordEnd = -1;

  for (let i = 0; i < characters.length; i++) {
    const ch = characters[i];
    if (ch === ' ' || ch === '\n' || ch === '\t') {
      if (currentWord.length > 0) {
        words.push({ word: currentWord, start: wordStart, end: wordEnd });
        currentWord = '';
        wordStart = -1;
      }
    } else {
      if (wordStart < 0) wordStart = starts[i];
      currentWord += ch;
      wordEnd = ends[i];
    }
  }
  // Flush last word
  if (currentWord.length > 0) {
    words.push({ word: currentWord, start: wordStart, end: wordEnd });
  }

  return words;
}

/**
 * Make an HTTPS POST request and return the parsed JSON response.
 */
function httpsPost(path, headers, body) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: ELEVENLABS_BASE,
      port: 443,
      path,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...headers
      }
    };

    const req = https.request(options, (res) => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const raw = Buffer.concat(chunks).toString();
        if (res.statusCode >= 400) {
          reject(new Error(`ElevenLabs API error ${res.statusCode}: ${raw}`));
          return;
        }
        try {
          resolve(JSON.parse(raw));
        } catch (e) {
          reject(new Error(`Failed to parse ElevenLabs response: ${raw.substring(0, 200)}`));
        }
      });
    });

    req.on('error', reject);
    req.write(JSON.stringify(body));
    req.end();
  });
}

/**
 * Sleep for a given number of milliseconds.
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Synthesize a single text segment with ElevenLabs TTS.
 *
 * @param {string} text - Narration text to synthesize
 * @param {string} voiceId - ElevenLabs voice ID
 * @param {string} apiKey - ElevenLabs API key
 * @param {Object} [options]
 * @param {string} [options.modelId] - Model ID (default: eleven_multilingual_v2)
 * @param {number} [options.stability] - Voice stability 0-1 (default: 0.5)
 * @param {number} [options.similarityBoost] - Similarity boost 0-1 (default: 0.75)
 * @returns {Promise<{ audioBase64: string, wordTimestamps: Array<{ word: string, start: number, end: number }>, duration: number }>}
 */
async function synthesizeSegment(text, voiceId, apiKey, options = {}) {
  const {
    modelId = DEFAULT_MODEL,
    stability = 0.5,
    similarityBoost = 0.75
  } = options;

  const path = `/v1/text-to-speech/${voiceId}/with-timestamps`;
  const headers = { 'xi-api-key': apiKey };
  const body = {
    text,
    model_id: modelId,
    voice_settings: {
      stability,
      similarity_boost: similarityBoost
    }
  };

  let lastError;
  for (let attempt = 0; attempt <= RETRY_MAX; attempt++) {
    try {
      if (attempt > 0) {
        const delay = RETRY_DELAY_MS * Math.pow(2, attempt - 1);
        process.stderr.write(`  Retry ${attempt}/${RETRY_MAX} after ${delay}ms...\n`);
        await sleep(delay);
      }

      const response = await httpsPost(path, headers, body);

      if (!response.audio_base64) {
        throw new Error('Response missing audio_base64 field');
      }
      if (!response.alignment) {
        throw new Error('Response missing alignment field (timestamps not returned)');
      }

      const wordTimestamps = characterTimestampsToWords(response.alignment);
      const duration = wordTimestamps.length > 0
        ? wordTimestamps[wordTimestamps.length - 1].end
        : 0;

      return {
        audioBase64: response.audio_base64,
        wordTimestamps,
        duration
      };
    } catch (err) {
      lastError = err;
      if (err.message.includes('429') || err.message.includes('rate')) {
        continue; // Retry on rate limit
      }
      if (attempt < RETRY_MAX && err.message.includes('5')) {
        continue; // Retry on 5xx
      }
      throw err;
    }
  }
  throw lastError;
}

/**
 * Synthesize multiple segments sequentially with rate limiting.
 * Returns an array of results in the same order as the input segments.
 *
 * @param {Array<{ id: string, text: string }>} segments - Segments to synthesize
 * @param {string} voiceId - ElevenLabs voice ID
 * @param {string} apiKey - ElevenLabs API key
 * @param {Object} [options] - Same options as synthesizeSegment
 * @param {Function} [onProgress] - Called with (completedCount, totalCount, segmentId)
 * @param {Function} [onResult] - Called with (result) after each segment completes, for incremental saving
 * @returns {Promise<Array<{ id: string, audioBase64: string, wordTimestamps: Array, duration: number }>>}
 */
async function synthesizeAll(segments, voiceId, apiKey, options = {}, onProgress = null, onResult = null) {
  const results = [];

  for (let i = 0; i < segments.length; i++) {
    const seg = segments[i];
    process.stderr.write(`  Synthesizing segment ${i + 1}/${segments.length}: "${seg.text.substring(0, 50)}..."\n`);

    const result = await synthesizeSegment(seg.text, voiceId, apiKey, options);
    const fullResult = { id: seg.id, ...result };
    results.push(fullResult);

    if (onProgress) onProgress(i + 1, segments.length, seg.id);
    if (onResult) onResult(fullResult);

    // Rate limit gap between segments
    if (i < segments.length - 1) {
      await sleep(SEGMENT_GAP_MS);
    }
  }

  return results;
}

module.exports = {
  synthesizeSegment,
  synthesizeAll,
  characterTimestampsToWords
};
