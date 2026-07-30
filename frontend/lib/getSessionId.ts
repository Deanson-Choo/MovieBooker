import { v4 as uuidv4 } from 'uuid';

export default function getSessionId(): string {
    const sessionId = sessionStorage.getItem('sessionId');
    console.log("Retrieved sessionId from sessionStorage:", sessionId);
    if (!sessionId) {
        const newSessionId = uuidv4();
        sessionStorage.setItem('sessionId', newSessionId);
        return newSessionId;
    }
    return sessionId;
}